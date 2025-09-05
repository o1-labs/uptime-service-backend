package main

import (
	. "block_producers_uptime/delegation_backend"
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
)

func main() {
	// Setup logging
	logging.SetupLogging(logging.Config{
		Format: logging.JSONOutput,
		Stderr: true,
		Stdout: false,
		Level:  logging.LevelDebug,
		File:   "",
	})
	log := logging.Logger("delegation backend")
	log.Infof("delegation backend has the following logging subsystems active: %v", logging.GetSubsystems())

	// Context and app initialization
	ctx := context.Background()
	appCfg := LoadEnv(log)
	app := new(App)
	app.IsReady = false
	app.Log = log
	awsctx := AwsContext{}
	kc := KeyspaceContext{}
	pctx := PostgreSQLContext{}
	app.VerifySignatureDisabled = appCfg.VerifySignatureDisabled
	if app.VerifySignatureDisabled {
		log.Warnf("Signature verification is disabled, it is not recommended to run the delegation backend in this mode!")
	}
	app.NetworkId = NetworkId(appCfg.NetworkName)

	// Storage backend setup
	if appCfg.Aws != nil {
		log.Infof("storage backend: AWS S3")
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(appCfg.Aws.Region))
		if err != nil {
			log.Fatalf("Error loading AWS configuration: %v", err)
		}
		client := s3.NewFromConfig(awsCfg)
		awsctx = AwsContext{Client: client, BucketName: aws.String(GetAWSBucketName(appCfg)), Prefix: appCfg.NetworkName, Context: ctx, Log: log}

	}

	if appCfg.AwsKeyspaces != nil {
		log.Infof("storage backend: AWS Keyspaces")
		session, err := InitializeKeyspaceSession(appCfg.AwsKeyspaces)
		if err != nil {
			log.Fatalf("Error initializing Keyspace session: %v", err)
		}
		defer session.Close()

		kc = KeyspaceContext{
			Session:  session,
			Keyspace: appCfg.AwsKeyspaces.Keyspace,
			Context:  ctx,
			Log:      log,
		}

	}

	if appCfg.LocalFileSystem != nil {
		log.Infof("storage backend: Local File System")
	}

	if appCfg.PostgreSQL != nil {
		log.Infof("storage backend: PostgreSQL")
		db, err := NewPostgreSQL(appCfg.PostgreSQL)
		if err != nil {
			log.Fatalf("Error initializing PostgreSQL: %v", err)
		}
		defer db.Close()

		pctx = PostgreSQLContext{
			DB:  db,
			Log: log,
		}
	}

	app.Save = func(objs ObjectsToSave) {
		if appCfg.Aws != nil {
			awsctx.S3Save(objs)
		}
		if appCfg.AwsKeyspaces != nil {
			kc.KeyspaceSave(objs)
		}
		if appCfg.PostgreSQL != nil {
			pctx.PostgreSQLSave(objs)
		}
		if appCfg.LocalFileSystem != nil {
			LocalFileSystemSave(objs, appCfg.LocalFileSystem.Path, log)
		}
	}

	if appCfg.Aws == nil && appCfg.LocalFileSystem == nil && appCfg.AwsKeyspaces == nil {
		log.Fatal("No storage backend configured!")
	}

	// App other configurations
	app.Now = func() time.Time { return time.Now() }
	requestsPerPkHourly := SetRequestsPerPkHourly(log)
	app.SubmitCounter = NewAttemptCounter(requestsPerPkHourly)
	log.Infof("Max requests per pk hourly: %v", requestsPerPkHourly)

	// HTTP handlers setup
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte("delegation backend service"))
	})
	http.Handle("/v1/submit", app.NewSubmitH())

	// Health check endpoint
	http.HandleFunc("/health", HealthHandler(func() bool {
		return app.IsReady
	}))

	// Sheets service and whitelist loop
	app.WhitelistDisabled = appCfg.DelegationWhitelistDisabled
	if app.WhitelistDisabled {
		log.Infof("Delegation whitelist is disabled")
	} else {
		sheetsService, err2 := sheets.NewService(ctx, option.WithScopes(sheets.SpreadsheetsReadonlyScope))
		if err2 != nil {
			log.Fatalf("Error creating Sheets service: %v", err2)
		}
		initWl, err := RetrieveWhitelist(sheetsService, log, appCfg, 1)
		if err != nil {
			log.Fatalf("Failed to initialize whitelist: %v", err)
		}
		wlMvar := new(WhitelistMVar)
		wlMvar.Replace(&initWl)
		app.Whitelist = wlMvar
		log.Infof("Delegation whitelist is enabled")
		go func() {
			for {
				time.Sleep(SetWhitelistRefreshInterval(log))
				wl, err := RetrieveWhitelist(sheetsService, log, appCfg, 10)
				if err != nil {
					log.Errorf("Failed to refresh delegation whitelist, using previous one, error: %v", err)
				} else {
					wlMvar.Replace(&wl)
					log.Infof("Delegation whitelist refreshed, number of BPs: %v", len(wl))
				}
			}
		}()
	}

	// Start server
	app.IsReady = true
	log.Infof("Server ready and listening on %s", DELEGATION_BACKEND_LISTEN_TO)
	log.Infof("Available endpoints: / (root), /v1/submit (submissions), /health (health check)")
	log.Fatal(http.ListenAndServe(DELEGATION_BACKEND_LISTEN_TO, nil))
}
