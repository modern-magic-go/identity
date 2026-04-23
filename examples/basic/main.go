package main

import (
	"context"
	"database/sql"
	"bufio"
	"fmt"
	"log"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"github.com/modern-magic-go/identity"
	mysqlstore "github.com/modern-magic-go/identity/examples/basic/adapters/mysqlstore"
	recache "github.com/modern-magic-go/identity/examples/basic/adapters/rediscache"
	"github.com/modern-magic-go/identity/usecase"
)

type config struct {
	MySQLDSN     string
	RedisAddr    string
	RedisUser    string
	RedisPassword string
	RedisDB      int
	RedisTTL     time.Duration
	SeedDemo     bool
	Realm        string
	Identifier   string
	Password     string
	SubjectNo    string
	SubjectType  string
	UseRedis     bool
}

func main() {
	ctx := context.Background()
	if err := loadDotEnv(); err != nil {
		log.Printf("load .env: %v", err)
	}
	cfg := loadConfig()

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping mysql: %v", err)
	}
	if err := mysqlstore.EnsureSchema(ctx, db); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	mysqlSubjects := mysqlstore.NewSubjectRepository(db)
	mysqlIdentities := mysqlstore.NewIdentityRepository(db)
	mysqlCredentials := mysqlstore.NewPasswordCredentialRepository(db)

	var subjects identity.SubjectRepository = mysqlSubjects
	var identities identity.IdentityRepository = mysqlIdentities
	var credentials identity.PasswordCredentialRepository = mysqlCredentials

	if cfg.UseRedis {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Username: cfg.RedisUser,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Fatalf("ping redis: %v", err)
		}
		defer rdb.Close()

		subjects = recache.NewSubjectRepository(subjects, rdb, cfg.RedisTTL)
		identities = recache.NewIdentityRepository(identities, rdb, cfg.RedisTTL)
		credentials = recache.NewPasswordCredentialRepository(credentials, rdb, cfg.RedisTTL)
	}

	svc, err := usecase.NewService(usecase.Config{
		Subjects:         subjects,
		Identities:       identities,
		Credentials:      credentials,
		PasswordVerifier:  demoPasswordVerifier{},
		Clock:            fixedClock{now: time.Now().UTC()},
		IDGenerator:      demoIDGenerator{},
	})
	if err != nil {
		log.Fatalf("new service: %v", err)
	}

	if cfg.SeedDemo {
		if err := seedDemoData(ctx, mysqlSubjects, mysqlIdentities, mysqlCredentials, cfg); err != nil {
			log.Fatalf("seed demo data: %v", err)
		}
	}

	result, err := svc.PasswordLogin(ctx, usecase.PasswordLoginInput{
		Realm:      cfg.Realm,
		Identifier: cfg.Identifier,
		Password:   cfg.Password,
	})
	if err != nil {
		log.Fatalf("password login: %v", err)
	}

	fmt.Printf("login success: subject_id=%d subject_no=%s subject_type=%s realm=%s\n",
		result.Subject.SubjectID,
		result.Subject.SubjectNo,
		result.Subject.SubjectType,
		result.Subject.Realm,
	)
}

func loadDotEnv() error {
	for _, path := range []string{
		filepath.Join("examples", "basic", ".env"),
		".env",
	} {
		if err := loadEnvFile(path); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return os.ErrNotExist
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return applyEnvFile(f)
}

func applyEnvFile(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.TrimPrefix(value, "export ")
		if len(value) >= 2 {
			if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		if key != "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func loadConfig() config {
	redisTTL := mustDuration(getenv("REDIS_TTL", "5m"))
	redisAddr := strings.TrimSpace(getenv("REDIS_ADDR", ""))
	return config{
		MySQLDSN:      getenv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/identity?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci"),
		RedisAddr:     redisAddr,
		RedisUser:     getenv("REDIS_USER", ""),
		RedisPassword: getenv("REDIS_PASSWORD", ""),
		RedisDB:       mustInt(getenv("REDIS_DB", "0")),
		RedisTTL:      redisTTL,
		SeedDemo:      mustBool(getenv("DEMO_SEED", "true")),
		Realm:         getenv("DEMO_REALM", "admin"),
		Identifier:    getenv("DEMO_IDENTIFIER", "admin_user"),
		Password:      getenv("DEMO_PASSWORD", "123456"),
		SubjectNo:     getenv("DEMO_SUBJECT_NO", "sub-1"),
		SubjectType:   getenv("DEMO_SUBJECT_TYPE", "admin"),
		UseRedis:      redisAddr != "",
	}
}

func seedDemoData(ctx context.Context, subjects *mysqlstore.SubjectRepository, identities *mysqlstore.IdentityRepository, credentials *mysqlstore.PasswordCredentialRepository, cfg config) error {
	now := time.Now().UTC()
	subject := &identity.Subject{
		ID:          1,
		SubjectNo:   cfg.SubjectNo,
		SubjectType: cfg.SubjectType,
		Realm:       cfg.Realm,
		Status:      identity.SubjectStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := subjects.Save(ctx, subject); err != nil {
		return err
	}
	ident := &identity.Identity{
		ID:            10,
		SubjectID:     subject.ID,
		Realm:         cfg.Realm,
		Provider:      "password",
		IdentityType:  "username",
		ProviderAppID: "",
		Identifier:    cfg.Identifier,
		UnionID:       "",
		Status:        identity.IdentityStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := identities.Save(ctx, ident); err != nil {
		return err
	}
	credential := &identity.PasswordCredential{
		ID:               20,
		SubjectID:        subject.ID,
		Realm:            cfg.Realm,
		PasswordHash:     "hash:" + cfg.Password,
		PasswordAlgo:     "demo",
		PasswordVersion:  1,
		NeedReset:        false,
		FailedCount:      0,
		Status:           identity.CredentialStatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return credentials.Save(ctx, credential)
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type demoIDGenerator struct{}

func (demoIDGenerator) NewID(prefix string) (string, error) {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 10), nil
}

type demoPasswordVerifier struct{}

func (demoPasswordVerifier) VerifyPassword(_ context.Context, plainPassword, storedHash, _ string) error {
	if storedHash != "hash:"+plainPassword {
		return fmt.Errorf("password mismatch")
	}
	return nil
}

func getenv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func mustBool(value string) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}

func mustInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func mustDuration(value string) time.Duration {
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 5 * time.Minute
	}
	return parsed
}
