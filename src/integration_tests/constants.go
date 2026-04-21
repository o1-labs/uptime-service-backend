package integration_tests

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	TEST_DATA_FOLDER = "../../test/integration"

	GENESIS_FILE  = TEST_DATA_FOLDER + "/topology/genesis_ledger.json"
	TOPOLOGY_FILE = TEST_DATA_FOLDER + "/topology/topology.json"

	UPTIME_SERVICE_CONFIG_DIR = TEST_DATA_FOLDER + "/topology/uptime_service_config"
	APP_CONFIG_FILE           = UPTIME_SERVICE_CONFIG_DIR + "/app_config.json"

	TIMEOUT_IN_S = 1500

	// AWS Keyspaces
	DATABASE_MIGRATION_DIR   = "../../database/migrations"
	AWS_SSL_CERTIFICATE_PATH = "../../database/cert/sf-class2-root.crt"
)

func getDirFiles(dir string, suffix string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var filteredFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), suffix) {
			absolutePath := dir + "/" + f.Name()
			filteredFiles = append(filteredFiles, absolutePath)
		}
	}

	return filteredFiles, nil
}

func getGpgFiles(dir string) ([]string, error) {
	return getDirFiles(dir, ".gpg")
}

func getJsonFiles(dir string) ([]string, error) {
	return getDirFiles(dir, ".json")
}

// copy app_config_*.json to app_config.json
func setAppConfig(config_type string) error {
	conf_file := "/app_config_" + config_type + ".json"
	log.Printf("Setting %s as %s...\n", conf_file, APP_CONFIG_FILE)

	err := copyFile(UPTIME_SERVICE_CONFIG_DIR+conf_file, APP_CONFIG_FILE)
	if err != nil {
		return fmt.Errorf("Error copying %s: %s\n", APP_CONFIG_FILE, err)
	}

	return nil
}

func encodeUptimeServiceConf() error {
	fixturesSecret := os.Getenv("UPTIME_SERVICE_SECRET")
	if fixturesSecret == "" {
		return fmt.Errorf("Error: UPTIME_SERVICE_SECRET environment variable not set")
	}

	jsonFiles, err := getJsonFiles(UPTIME_SERVICE_CONFIG_DIR)
	if err != nil {
		return err
	}
	for _, json_file := range jsonFiles {
		log.Printf(">> Encoding %s...\n", json_file)

		// Construct the gpg command
		cmd := exec.Command(
			"gpg",
			"--pinentry-mode", "loopback",
			"--passphrase", fixturesSecret,
			"--symmetric",
			"--output", fmt.Sprintf("%s.gpg", json_file),
			json_file,
		)

		// Execute and get output
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error encoding %s: %s\n", json_file, err)
		}

		log.Println(string(out))
	}

	return nil
}

func decodeUptimeServiceConf() error {
	fixturesSecret := os.Getenv("UPTIME_SERVICE_SECRET")
	if fixturesSecret == "" {
		return fmt.Errorf("Error: UPTIME_SERVICE_SECRET environment variable not set")
	}

	gpgFiles, err := getGpgFiles(UPTIME_SERVICE_CONFIG_DIR)
	if err != nil {
		return err
	}

	for _, gpg_file := range gpgFiles {
		json_file := strings.TrimSuffix(gpg_file, ".gpg")
		// skip if file exists
		if _, err := os.Stat(json_file); err == nil {
			log.Printf(">> Skipping decoding %s... JSON file already exists.\n", gpg_file)
			continue
		}

		log.Printf(">> Decoding %s...\n", gpg_file)

		// Construct the gpg command
		cmd := exec.Command(
			"gpg",
			"--pinentry-mode", "loopback",
			"--yes",
			"--passphrase", fixturesSecret,
			"--output", json_file,
			"--decrypt", gpg_file,
		)

		// Execute and get output
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error decoding %s: %s\n", gpg_file, err)
		}

		log.Println(string(out))
	}

	return nil
}
