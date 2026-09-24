package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"mall/admin_core/bootstrap"
	_ "mall/admin_core/docs"
	"mall/admin_core/internal/auth"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/pkg/logger"
	pkgversion "mall/admin_core/pkg/version"
	"mall/admin_core/routers"
)

// @title Mall Admin Core API
// @version 0.1.0
// @description Simple commerce administration backend.
// @host localhost:8100
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	var (
		initFlag        = flag.Bool("init", false, "run database migrations and seeds (clean install), then exit")
		seedFlag        = flag.Bool("seed", false, "run database seeds, then exit")
		alignFlag       = flag.Bool("align", false, "align existing database schema_migrations to baseline and run seeds, then exit")
		createAdminFlag = flag.Bool("create-admin", false, "create the initial super administrator, then exit")
		versionFlag     = flag.Bool("version", false, "print version information and exit")
		adminUsername   = flag.String("username", "", "admin username (default reads from ADMIN_USERNAME env)")
		adminPassword   = flag.String("password", "", "admin password (default reads from ADMIN_PASSWORD env)")
		adminNickname   = flag.String("nickname", "", "admin nickname (default reads from ADMIN_NICKNAME env)")
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of mall admin core:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  go run . [command|options]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  init          Run database migrations and seeds (clean install), then exit\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  seed          Run database seeds, then exit\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  align         Align existing database schema_migrations to baseline and run seeds, then exit\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  create-admin  Create the initial super administrator, then exit\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  version       Print version information and exit\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	logCfg := logger.LoadConfigFromEnv()
	logInitErr := logger.Init(logCfg)
	if logInitErr != nil {
		log.Printf("failed to initialize logger: %v", logInitErr)
	}
	defer logger.Sync()

	if *versionFlag {
		info := pkgversion.Get()
		fmt.Printf("GT-Shop v%s\nBuild %s\nCommit %s\n", info.Version, info.BuildTime, info.GitCommit)
		return
	}

	action := ""
	if *initFlag {
		action = "init"
	} else if *seedFlag {
		action = "seed"
	} else if *alignFlag {
		action = "align"
	} else if *createAdminFlag {
		action = "create-admin"
	} else if flag.NArg() > 0 {
		switch flag.Arg(0) {
		case "init":
			action = "init"
		case "seed":
			action = "seed"
		case "align":
			action = "align"
		case "create-admin":
			action = "create-admin"
		case "version":
			info := pkgversion.Get()
			fmt.Printf("GT-Shop v%s\nBuild %s\nCommit %s\n", info.Version, info.BuildTime, info.GitCommit)
			return
		default:
			flag.Usage()
			os.Exit(1)
		}
	}

	config := bootstrap.LoadConfig()

	if action != "" {
		if !config.DatabaseSet {
			log.Fatalf("DATABASE_DSN is not configured; cannot execute database %s", action)
		}
		db, err := bootstrap.NewDB(context.Background(), config)
		if err != nil {
			log.Fatalf("failed to connect to database for %s: %v", action, err)
		}
		sqlDB, dbErr := db.DB()
		if dbErr != nil {
			log.Fatalf("database handle is unavailable: %v", dbErr)
		}
		defer sqlDB.Close()

		ctx := context.Background()
		switch action {
		case "init":
			log.Print("starting database initialization (migrations + seeds)...")
			if err := bootstrap.RunInit(ctx, db); err != nil {
				log.Fatalf("database initialization failed: %v", err)
			}
			log.Print("database initialization completed successfully")
			return
		case "seed":
			log.Print("starting database seeding...")
			if err := bootstrap.RunSeeds(ctx, db); err != nil {
				log.Fatalf("database seeding failed: %v", err)
			}
			log.Print("database seeding completed successfully")
			return
		case "align":
			log.Print("starting database baseline alignment...")
			if err := bootstrap.AlignBaseline(ctx, db); err != nil {
				log.Fatalf("database baseline alignment failed: %v", err)
			}
			log.Print("database baseline alignment completed successfully")
			return
		case "create-admin":
			username := strings.TrimSpace(*adminUsername)
			if username == "" {
				username = strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
			}
			if username == "" {
				log.Fatal("ADMIN_USERNAME must be provided (via ADMIN_USERNAME env or -username flag)")
			}

			password := *adminPassword
			if password == "" {
				password = os.Getenv("ADMIN_PASSWORD")
			}
			if password == "" {
				log.Fatal("ADMIN_PASSWORD must be explicitly provided (via ADMIN_PASSWORD env or -password flag)")
			}

			nickname := strings.TrimSpace(*adminNickname)
			if nickname == "" {
				nickname = strings.TrimSpace(os.Getenv("ADMIN_NICKNAME"))
			}

			enforcer, err := bootstrap.NewCasbinEnforcer(db)
			if err != nil {
				log.Fatalf("failed to initialize casbin enforcer: %v", err)
			}
			authorizer := auth.NewService(db, enforcer, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)

			log.Printf("creating super admin user %q...", username)
			createdAdmin, err := bootstrap.CreateAdmin(ctx, db, authorizer, username, password, nickname)
			if err != nil {
				log.Fatalf("failed to create admin: %v", err)
			}
			log.Printf("super admin %q (ID: %d) created successfully with role super_admin", createdAdmin.Username, createdAdmin.ID)
			return
		}
	}
	if config.JWTSecret == "" {
		log.Print("AUTH_JWT_SECRET is not configured; admin login is unavailable")
	}

	db, err := bootstrap.NewDB(context.Background(), config)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	var authorization *auth.Service
	casbinReady := true
	if db != nil {
		sqlDB, dbErr := db.DB()
		if dbErr != nil {
			log.Printf("database handle is unavailable; starting without database: %v", dbErr)
		} else {
			defer sqlDB.Close()
		}
		if config.AutoMigrate {
			if err := bootstrap.RunMigrations(context.Background(), db); err != nil {
				log.Printf("migration failed: %v", err)
				casbinReady = false
			}
		}
		if casbinReady {
			enforcer, err := bootstrap.NewCasbinEnforcer(db)
			if err != nil {
				log.Printf("casbin initialization failed: %v", err)
			} else {
				authorization = auth.NewService(db, enforcer, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)
			}
		}
	}
	if authorization == nil {
		authorization = auth.NewService(db, nil, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)
	}

	cacheInstance := bootstrap.NewCache(context.Background(), config)
	defer cacheInstance.Close()

	router := routers.NewRouter(config, db, authorization, cacheInstance)
	if db != nil && casbinReady {
		workerCtx, stopWorker := context.WithCancel(context.Background())
		defer stopWorker()
		go miniapppayment.NewService(db).RunTimeoutWorker(workerCtx,
			miniapppayment.DefaultOrderExpiry, miniapppayment.DefaultSweepInterval)
		go orderaftersale.NewService(db, nil, wechatpay.RefundProvider{Store: wechatpay.Store{DB: db}}, cacheInstance).RunRefundWorker(workerCtx)
	}
	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()
	startupInfo := bootstrap.CollectStartupInfo(context.Background(), config, db, cacheInstance, logCfg, logInitErr)
	startupInfo.Listen = listener.Addr().String()
	bootstrap.PrintStartupInfo(startupInfo)
	if err := http.Serve(listener, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
