package configparser

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var gConfig *Configuration

const (
	CONFIG_FILEPATH         = "/tmp/configparser_test.ini"
	CONFIG_FILEPATH_SHA     = "7594b11800abe3dbc4b82d3c9ccab8c6160d6c8e"
	CONFIG_NEW_FILEPATH     = "/tmp/configparser_test_new.ini"
	CONFIG_NEW_FILEPATH_SHA = "727325676a4e15ed451ba91a2ac9cc0f8db65606"

	SECTION_NAME_1     = "MYSQLD DEFAULT"
	SECTION_NAME_2     = "MONGODB"
	SECTION_NAME_3     = "NDB_MGMD DEFAULT"
	SECTION_NAME_REGEX = "webservers$"

	KEY_1 = "TotalSendBufferMemory"
	KEY_2 = "DefaultOperationRedoProblemAction"
	KEY_3 = "innodb_buffer_pool_size"
	KEY_4 = "innodb_buffer_pool_instances"

	KEY_5 = "datadir"
	KEY_6 = "smallfiles"

	CONFIG_FILE_CONTENT = `wsrep_provider_options="gcache.size=128M; evs.keepalive_period=PT3S; evs.inactive_check_period=PT10S; evs.suspect_timeout=PT30S; evs.inactive_timeout=PT1M; evs.consensus_timeout=PT1M; evs.send_window=1024; evs.user_send_window=512;"

SendBufferMemory = 20M
ReceiveBufferMemory = 20M

[dc1.webservers]
10.10.10.10
20.20.20.20
dc1.backup.local

[dc2.database]
30.30.30.30
40.40.40.40
dc2.standby.local

[dc2.webservers]
30.30.30.30
40.40.40.40

[TCP DEFAULT]
#SendBufferMemory=20M
#ReceiveBufferMemory=20M

[NDBD DEFAULT]
NoOfReplicas=2
DataDir=/data/mysql/cluster/dev
FileSystemPath=/data/mysql/cluster/dev
#FileSystemPathDD=
#FileSystemPathDataFiles=
#FileSystemPathUndoFiles=
#BackupDataDir=
#InitialLogFileGroup=name=lg1;undo_buffer_size:64M;undo1.log:64M
#InitialTablespace=name=ts1;extent_size:1M;data1.dat:256M;data2.dat:256M

DataMemory:256M
IndexMemory:32M
DiskPageBufferMemory:64M
SharedGlobalMemory=128M
RedoBuffer=48M
TotalSendBufferMemory=20M

LockPagesInMainMemory=1
Numa=0

RealtimeScheduler=1
MaxNoOfExecutionThreads=4
#LockExecuteThreadToCPU=
#LockMaintThreadsToCPU=
DiskIOThreadPool=2

BuildIndexThreads=2
TwoPassInitialNodeRestartCopy=1

DiskCheckpointSpeedInRestart=100M
DiskCheckpointSpeed=10M

FragmentLogFileSize=256M
NoOfFragmentLogFiles=6
InitFragmentLogFiles=SPARSE

ODirect=1
;CompressedBackup=0
;CompressedLCP=0
Diskless=0

TimeBetweenLocalCheckpoints=20
TimeBetweenGlobalCheckpoints=2000
TimeBetweenEpochs=100
;This parameter defines a timeout for synchronization epochs for MySQL Cluster Replication. If a node fails to participate in a global checkpoint within the time determined by this parameter, the node is shut down
#TimeBetweenEpochsTimeout=4000
# Set in production
#TimeBetweenInactiveTransactionAbortCheck=1000
#TransactionDeadlockDetectionTimeout=1200
#TransactionInactiveTimeout=0
# Might need to increase initial check for large data memory allocations
#TimeBetweenWatchDogCheckInitial = 6000
#TimeBetweenWatchDogCheck= 6000

MaxNoOfConcurrentOperations=250000
MaxNoOfConcurrentScans=500
#MaxNoOfLocalScans=2048

#MaxNoOfConcurrentScans=256 (2-500)
#MaxNoOfLocalScans=numOfDataNodes*MaxNoOfConcurrentScans
# 1-992
#BatchSizePerLocalScan=900
#MaxParallelScansPerFragment=256 (1-1G)

# % of max value
StringMemory=25
MaxNoOfTables=2048
MaxNoOfOrderedIndexes=1024
MaxNoOfUniqueHashIndexes=1024
MaxNoOfAttributes=8192
MaxNoOfTriggers=8192

#MemReportFrequency=10
StartupStatusReportFrequency=10

### Params for setting logging
LogLevelStartup=15
LogLevelShutdown=15
LogLevelCheckpoint=8
LogLevelNodeRestart=15
LogLevelCongestion=15
LogLevelStatistic=15

### Params for increasing Disk throughput
BackupDataBufferSize=16M
BackupLogBufferSize=16M
BackupMemory=32M
#If BackupDataBufferSize and BackupLogBufferSize taken together exceed the default value for BackupMemory, then this parameter must be set explicitly in the config.ini file to their sum.
BackupWriteSize=256K
BackupMaxWriteSize=1M
BackupReportFrequency=10

### CGE 6.3 - REALTIME EXTENSIONS
#RealTimeScheduler=1
#SchedulerExecutionTimer=80
#SchedulerSpinTimer=40

RedoOverCommitCounter=3
RedoOverCommitLimit=20

StartFailRetryDelay=0
MaxStartFailRetries=3

[NDB_MGMD DEFAULT]
PortNumber=1186
DataDir=/data/mysql/cluster/dev
#MaxNoOfSavedEvents=100
TotalSendBufferMemory=4M

[NDB_MGMD]
NodeId=1
HostName=localhost
PortNumber=1186
ArbitrationRank=1

#[NDB_MGMD]
#NodeId=2
#HostName=localhost
#PortNumber=1187
#ArbitrationRank=1

[NDBD]
NodeId=10
HostName=localhost
#HeartbeatOrder=10

[NDBD]
NodeId=11
HostName=localhost
#HeartbeatOrder=20

[NDBD]
NodeId=12
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

[NDBD]
NodeId=13
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

[NDBD]
NodeId=14
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

[NDBD]
NodeId=15
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

#
# Note=The following can be MySQLD connections or
#      NDB API application connecting to the cluster
#
[MYSQLD DEFAULT]
TotalSendBufferMemory=10M
DefaultOperationRedoProblemAction=ABORT
#DefaultOperationRedoProblemAction=QUEUE
#BatchByteSize=32K (1024-1M)
# 1-992
#BatchSize=900
#MaxScanBatchSize=256K (32K-16M)
; this is another comment
[MYSQLD]
NodeId=100
HostName=localhost

[API]
NodeId=101
[API]
NodeId=102
[API]
NodeId=103
[API]
NodeId=104
[API]
NodeId=105
[API]
NodeId=106
[API]
NodeId=107
[API]
NodeId=108
[API]
NodeId=109
[API]
NodeId=110
[API]
NodeId=111

[API]
NodeId=200

[API]
NodeId=201
[API]
NodeId=202
[API]
NodeId=203
[API]
NodeId=204
[API]
NodeId=205
[API]
NodeId=206
[API]
NodeId=207
[API]
NodeId=208
[API]
NodeId=209
[API]
NodeId=210
[API]
NodeId=211

[API]
NodeId=212
[API]
NodeId=213
[API]
NodeId=214
[API]
NodeId=215
[API]
NodeId=216
[API]
NodeId=217
[API]
NodeId=218
[API]
NodeId=219
[API]
NodeId=220
[API]
NodeId=221
[API]
NodeId=222
`
)

func TestWriteTestConfigFile(t *testing.T) {
	t.Log("Writing test config to" + CONFIG_FILEPATH)
	f, err := os.Create(CONFIG_FILEPATH)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = f.Close()
	}()

	w := bufio.NewWriter(f)
	defer func() {
		err = w.Flush()
	}()

	w.WriteString(CONFIG_FILE_CONTENT)
}

func TestReadConfigFile(t *testing.T) {
	t.Log("Reading test config " + CONFIG_FILEPATH)

	var err error
	gConfig, err = Read(CONFIG_FILEPATH)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(gConfig)
}

func TestGetSection(t *testing.T) {
	s, err := getConfig().Section("NDBD DEFAULT")
	if err != nil {
		t.Error(err)
	}

	t.Log(s)
}

func TestGetSections(t *testing.T) {
	s, err := getConfig().Sections("NDBD")
	if err != nil {
		t.Error(err)
	}

	t.Log(s)
}

func TestSetNewValue(t *testing.T) {
	s, err := getConfig().Section(SECTION_NAME_1)
	if err != nil {
		t.Error(err)
	}

	t.Logf("%s=%s\n", KEY_1, s.ValueOf(KEY_1))
	oldValue := s.SetValueFor(KEY_1, "512M")
	t.Logf("New: %s=%s\n", KEY_1, s.ValueOf(KEY_1))
	if oldValue == s.ValueOf(KEY_1) {
		t.Error("Unable to change value for key " + s.ValueOf(KEY_1))
	}
}

func TestAddOption(t *testing.T) {
	s, err := getConfig().Section(SECTION_NAME_1)
	if err != nil {
		t.Error(err)
	}

	testAddOption(s, KEY_3, "128G", t)
	testAddOption(s, KEY_4, "16", t)

	testAddOption(s, KEY_3, "64G", t)
	testAddOption(s, KEY_4, "8", t)
}

func TestDeleteOption(t *testing.T) {
	s, err := getConfig().Section(SECTION_NAME_1)
	if err != nil {
		t.Error(err)
	}

	testDeleteOption(s, KEY_2, t)
}

func TestNotExistsOption(t *testing.T) {
	s, err := getConfig().Section(SECTION_NAME_1)
	if err != nil {
		t.Error(err)
	}

	if s.Exists("none_existing_key") {
		t.Error("none existing key found")
	}
}

func TestNewSection(t *testing.T) {
	s := getConfig().NewSection(SECTION_NAME_2)
	s.Add(KEY_5, "/var/lib/mongodb")
	s.Add(KEY_6, "true")
}

func TestGetNewSections(t *testing.T) {
	s, err := getConfig().Section(SECTION_NAME_2)
	if err != nil {
		t.Error(err)
	}
	if !s.Exists(KEY_5) {
		t.Error(KEY_5 + " does not exists")
	}

	if !s.Exists(KEY_6) {
		t.Error(KEY_6 + " does not exists")
	}

	t.Log(s)
}

func TestDeleteSection(t *testing.T) {
	c := getConfig()
	sections, err := c.Delete(SECTION_NAME_3)
	if err != nil {
		t.Error(err)
	}
	for _, s := range sections {
		t.Log(s)
	}
}

func TestFindSection(t *testing.T) {
	c := getConfig()
	sections, err := c.Find(SECTION_NAME_REGEX)
	if err != nil {
		t.Error(err)
	}
	for _, s := range sections {
		t.Log(s)
	}
}

func TestSaveNewConfigFile(t *testing.T) {
	c := getConfig()

	err := Save(c, CONFIG_NEW_FILEPATH)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSHA(t *testing.T) {
	out, err := exec.Command("shasum", CONFIG_NEW_FILEPATH).Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.Split(string(out), " ")
	t.Logf("%v=%v", sha[0], CONFIG_NEW_FILEPATH_SHA)
	if sha[0] != CONFIG_NEW_FILEPATH_SHA {
		t.Error(CONFIG_NEW_FILEPATH + " shasum doees not match!")
	}
}

func getConfig() *Configuration {
	if gConfig == nil {
		log.Println("No configuration instance!")
		os.Exit(1)
	}
	return gConfig
}

func testAddOption(s *Section, name string, value string, t *testing.T) {
	oldValue := s.Add(name, value)
	t.Logf("%s=%s, old value: %s\n", name, s.ValueOf(name), oldValue)
	if oldValue == s.ValueOf(name) {
		t.Error("Unable to change value for key " + s.ValueOf(name))
	}
}

func testDeleteOption(s *Section, name string, t *testing.T) {
	oldValue := s.Delete(name)
	t.Logf("%s=%s\n", name, oldValue)
}
