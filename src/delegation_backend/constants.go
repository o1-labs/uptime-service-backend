package delegation_backend

import (
	"os"
	"strconv"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

const MAX_SUBMIT_PAYLOAD_SIZE = 50000000 // max payload size in bytes
const DELEGATION_BACKEND_LISTEN_TO = ":8080"
const TIME_DIFF_DELTA time.Duration = -5 * 60 * 1000000000 // -5m
const WHITELIST_REFRESH_INTERVAL = 10 * 60 * 1000000000    // 10m
const ONLINE_KEEP_INTERVAL = 20 * time.Minute

var PK_PREFIX = [...]byte{1, 1}
var SIG_PREFIX = [...]byte{1}
var BLOCK_HASH_PREFIX = [...]byte{1}
var MAX_BLOCK_SIZE = 1000000 // (1MB) max block size in bytes for Cassandra, blocks larger than this size will be stored in S3 only

func NetworkId(networkName string) uint8 {
	if networkName == "mainnet" {
		return 1
	}
	return 0
}

func SetWhitelistRefreshInterval(log logging.StandardLogger) time.Duration {
	var defaultValue time.Duration = time.Duration(10) * time.Minute
	var whitelistRefreshInterval time.Duration

	envVarValue, exists := os.LookupEnv("DELEGATION_WHITELIST_REFRESH_INTERVAL")
	if exists {
		minutes, err := strconv.Atoi(envVarValue)
		if err != nil {
			log.Warnf("Error parsing DELEGATION_WHITELIST_REFRESH_INTERVAL, falling back to default value: %v, error: %v", defaultValue, err)
			whitelistRefreshInterval = defaultValue
		}
		whitelistRefreshInterval = time.Duration(minutes) * time.Minute
	} else {
		whitelistRefreshInterval = defaultValue
	}
	log.Infof("Delegation whitelist refresh interval: %v", whitelistRefreshInterval)
	return whitelistRefreshInterval
}

func SetRequestsPerPkHourly(log logging.StandardLogger) int {
	var defaultValue = 120
	var requestsPerPkHourly int
	var err error

	envVarValue, exists := os.LookupEnv("REQUESTS_PER_PK_HOURLY")
	if exists {
		requestsPerPkHourly, err = strconv.Atoi(envVarValue)
		if err != nil {
			log.Warnf("Error parsing REQUESTS_PER_PK_HOURLY, falling back to default value: %v, error: %v", defaultValue, err)
			requestsPerPkHourly = defaultValue
		}
	} else {
		requestsPerPkHourly = defaultValue
	}
	return requestsPerPkHourly
}

const PK_LENGTH = 33  // one field element (32B) + 1 bit (encoded as full byte)
const SIG_LENGTH = 64 // one field element (32B) and one scalar (32B)

// we use state hash code here, although it's not state hash
const BASE58CHECK_VERSION_BLOCK_HASH byte = 0x10
const BASE58CHECK_VERSION_PK byte = 0xCB
const BASE58CHECK_VERSION_SIG byte = 0x9A
