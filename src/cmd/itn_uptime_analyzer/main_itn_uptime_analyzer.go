package main

import (
	dg "block_producers_uptime/delegation_backend"
	itn "block_producers_uptime/itn_uptime_analyzer"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	logging "github.com/ipfs/go-log/v2"
)

func main() {
    // Set up sync period of type int representing minutes
    syncPeriod := 15

    // Setting up logging for application
    logging.SetupLogging(logging.Config{
        Format: logging.JSONOutput,
        Stderr: true,
        Stdout: false,
        Level:  logging.LevelDebug,
        File:   "",
    })
    log := logging.Logger("itn availability script")

    // Empty context object and initializing memory for application
    ctx := context.Background()

    // Load environment variables
    appCfg := itn.LoadEnv(log)

    awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(appCfg.Aws.Region))
    if err != nil {
        log.Fatalf("Error loading AWS configuration: %v\n", err)
    }

	var outputFile *os.File
	switch {
		case appCfg.Output.Local != "":
		    outputFile, err = os.Create(appCfg.Output.Local)

		// AppConfig already ensures that if S3 key is specified, bucket name is also specified.
		case appCfg.Output.S3Key != "":
		    outputFile, err = os.CreateTemp("", "itn_uptime_*.csv")
	}
	if err != nil {
		log.Fatalf("Error creating output file: %v", err)
	}
	if outputFile != nil {
		defer outputFile.Close()
	}

	var output func(string)
	switch {
		case appCfg.Output.Stdout && outputFile != nil:
		    output = func(msg string) {
				print(msg)
				outputFile.WriteString(msg)
			}
		case appCfg.Output.Stdout:
		    output = func(msg string) { print(msg) }
		case outputFile != nil:
		    output = func(msg string) {
				outputFile.WriteString(msg)
			}
		default:
		    log.Fatalf("No output specified!")
	}

    app := new(dg.App)
    app.Log = log
    client := s3.NewFromConfig(awsCfg, dg.S3OptionsFromEnv)

    awsctx := dg.AwsContext{Client: client, BucketName: aws.String(itn.GetBucketName(appCfg)), Prefix: appCfg.NetworkName, Context: ctx, Log: log}

    if appCfg.IgnoreIPs {
		output(fmt.Sprintf("Period start; %v\nPeriod end; %v\n",
			appCfg.Period.Start, appCfg.Period.End))
        output(fmt.Sprintf("Interval; %v\npublic key; uptime (%%)\n",
			appCfg.Period.Interval))
    } else {
        output(fmt.Sprintf("Period start; %v;\nPeriod end; %v;\n",
			appCfg.Period.Start, appCfg.Period.End))
        output(fmt.Sprintf("Interval; %v;\npublic key; public ip; uptime (%%)\n",
			appCfg.Period.Interval))
    }

    identities := itn.CreateIdentities(appCfg, awsctx, log)
    // Go over identities and calculate uptime
    for _, identity := range identities {
        identity.GetUptime(appCfg, awsctx, log, syncPeriod)
        if appCfg.IgnoreIPs {
            output(fmt.Sprintf("%s; %s\n",
				identity.PublicKey, *identity.Uptime))
        } else {
            output(fmt.Sprintf("%s; %s; %s\n",
				identity.PublicKey, identity.PublicIp, *identity.Uptime))
        }
    }

	// AppConfig already ensures that is S3Key is set, S3Bucket is set as well.
	if appCfg.Output.S3Key != "" {
		outputFileName := strings.Join([]string{*aws.String(appCfg.Output.S3Key), itn.OutputFileName(appCfg)}, "/")
		// AWS SDK is not smart enough to rewind the file, so we have to do it manually.
		_, err := outputFile.Seek(0, 0)
		if err != nil {
			log.Fatalf("File read error before upload: %v\n", err)
		} 
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(appCfg.Output.S3Bucket),
			Key:    &outputFileName,
			Body: outputFile,
		})
		if err != nil {
			log.Fatalf("Error uploading file to S3: %v\n", err)
		} 
	}
}
