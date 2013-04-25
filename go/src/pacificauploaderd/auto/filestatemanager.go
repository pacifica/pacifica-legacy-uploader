package auto

import (
	"errors"
	"fmt"
	"log"
	"pacificauploaderd/common"
	"os"
	"path/filepath"
	"platform"
	"sqlite"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	_DATABASE_CODE_VER         = 2
	_DATABASE_FILE_NAME string = "filestates.sdb"

	sqlGetFileState = "SELECT id, last_modified, digest, full_path, last_seen, bundle_id, bundle_file_id, " +
		"pass_off, user_name, rule_name, auto_added_to_bundle_time FROM file_states WHERE user_name=? AND " +
		"rule_name=? AND full_path=? LIMIT 1;"
	
	sqlGetFileStatesByBundleFileId = "SELECT id FROM file_states WHERE bundle_file_id=?;"

	sqlGetFileStateById = "SELECT id, last_modified, digest, full_path, last_seen, bundle_id, bundle_file_id, "+
		"pass_off, user_name, rule_name, auto_added_to_bundle_time FROM file_states WHERE id=?;"

)

type autoPassOffState int32

// These values are persisted to a database. Do not change the values or semantics, without being
// aware of the ramifications.
const (
	notSeenBefore  autoPassOffState = 0
	addingToBundle autoPassOffState = 1
	inBundle       autoPassOffState = 2
	done           autoPassOffState = 4
)

type fileState struct {
	id                    int64            //This fileState's id in the database.
	lastModified          int64            //Timestamp of Path's last modification.
	digest                string           //A message digest from hashing the file's content such as SHA1, etc.
	fullPath              string           //The full path to the file.
	lastSeen              int64            //The last time the file's path was checked for existence.
	bundleId              *int             //The id of the bundle this fileState is in.
	bundleFileId          *int             //The bundleFileId assigned to this file in bundleId.
	passOff               autoPassOffState //The state this fileState is in as it pertains to the auto manager.
	userName              string           //The user name associated with this fileState.
	ruleName              string           //The WatchRule name associated with this fileState.
	autoAddedToBundleTime int64            //The time when this fileState was added to an auto-submit bundle
}

// Represents the available unsubmitted bundle identifiers for a user.
type userBundles struct {
	user                  string
	autoSubmitBid         *int
	noAutoSubmitBid       *int
	autoSubmitLastTouched int64
}

// Manages stateDatabase creation, saves, additions, deletions, and cleanup.
type fileStateManager struct {
	db       *stateDatabase
	autoSave *time.Ticker
}

func fileStateInit() {
	//Set up necessary file system locations
	createDatabaseDir()
	filename := getDefaultDatabaseFilename()

	//Set the default fileStateManager
	fsm = newFileStateManager(filename)
}

func createDatabaseDir() {
	dir := common.StateDir
	os.MkdirAll(dir, 0700)
	if common.System && platform.PlatformGet() == platform.Windows {
		err := common.Cacls(dir, "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Panic("Failed to run cacls %v\n", err)
		}
	}
}

func getDefaultDatabaseFilename() string {
	path := filepath.Join(common.StateDir, _DATABASE_FILE_NAME)
	return path
}

// fileStates are saved to this database.
type stateDatabase struct {
	path  string
	conn  *sqlite.Conn
	mutex sync.Mutex
	getFileStateStmt *sqlite.Stmt
	getFileStatesByBundleFileIdStmt *sqlite.Stmt
	getFileStateByIdStmt *sqlite.Stmt
}

func newStateDatabase(filename string) *stateDatabase {
	self := new(stateDatabase)
	self.path = filename
	self.setupDatabase()

	if s, err := self.conn.Prepare(sqlGetFileState); err != nil {
		log.Fatalf("SQL statement %s prepare failed with error %v", sqlGetFileState, err)
	} else {
		self.getFileStateStmt = s
	}

	if s, err := self.conn.Prepare(sqlGetFileStatesByBundleFileId); err != nil {
		log.Fatalf("getFileStatesByBundleFileId SQL %s failed with error %v", sqlGetFileStatesByBundleFileId, err)
	} else {
		self.getFileStatesByBundleFileIdStmt = s
	}

	if s, err := self.conn.Prepare(sqlGetFileStateById); err != nil {
		log.Fatalf("SQL statement %s prepare failed with error %v", sqlGetFileStateById, err)
	} else {
		self.getFileStateByIdStmt = s
	}

	return self
}

func (self *stateDatabase) setupDatabase() {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	common.Dprintln("Entered stateDatabase.setupDatabase")

	//Open or create the database
	c, err := sqlite.Open(self.path)
	if err != nil {
		log.Fatalf("Could not open or create %s, error: %v\n", self.path, err)
	}
	self.conn = c

	//Turn on foreign key constraints
	err = self.conn.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	//Prepare to get the current database version
	s, err := self.conn.Prepare("select value from system where name = \"version\";")
	//The system table does not exist
	if err != nil && err.Error() == "SQL error or missing database: no such table: system" {
		err = self.conn.Exec("create table system(name string primary key, value string);")
		if err != nil {
			log.Fatalf("%v\n", err)
		}

		fileStatesSchema := "CREATE TABLE file_states(" +
			"id INTEGER PRIMARY KEY AUTOINCREMENT, " +
			"last_modified INTEGER NOT NULL, " +
			"digest STRING NULLABLE, " +
			"full_path STRING NOT NULL, " +
			"last_seen INTEGER NOT NULL, " +
			"bundle_id INTEGER NULLABLE, " +
			"bundle_file_id INTEGER NULLABLE, " +
			"pass_off INTEGER NOT NULL, " +
			"user_name STRING NOT NULL, " +
			"rule_name STRING NOT NULL, " +
			"auto_added_to_bundle_time INTEGER NOT NULL);"

		//Create the file_states table
		err = self.conn.Exec(fileStatesSchema)
		if err != nil {
			log.Fatalf("%v\nSQL: %s\n", err, fileStatesSchema)
		}

		userBundlesSchema := "CREATE TABLE user_bundles(" +
			"user STRING PRIMARY KEY, " +
			"auto_submit_bid INTEGER NULLABLE, " +
			"no_auto_submit_bid INTEGER NULLABLE, " +
			"auto_submit_last_touched INTEGER NOT NULL);"

		//Create the user_bundles table
		err = self.conn.Exec(userBundlesSchema)
		if err != nil {
			log.Fatalf("%v\nSQL: %s\n", err, userBundlesSchema)
		}

		schema := strings.Split("create index file_states_full_path ON file_states(full_path);" +
		                        "create index file_states_id ON file_states(id);", ";")
		for _, sql := range schema {
			if strings.TrimSpace(sql) == "" {
				continue
			}
			err = self.conn.Exec(sql + ";")
			if err != nil {
				log.Printf("%v\nSQL: %s\n", err, sql)
				os.Exit(1)
			}
		}

		//Set the database version
		err = self.conn.Exec("INSERT INTO system(name, value) VALUES(\"version\", \"" +
			strconv.Itoa(int(_DATABASE_CODE_VER)) + "\");")
		if err != nil {
			log.Fatalf("%v\n", err)
		}

		//Prepare statement to get the database version
		s, err = self.conn.Prepare("select value from system where name = \"version\";")
		if err != nil {
			log.Fatalf("%v\n", err)
		}
	}
	//Get the database version
	err = s.Exec()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	version := ""
	for {
		if !s.Next() {
			break
		}
		var value string
		err = s.Scan(&value)
		if err != nil {
			log.Fatalf("%v", err)
		}
		version = value
	}
	if version == "" {
		log.Fatalf("Failed to get a version from database.\n")
	}
	ver, err := strconv.ParseInt(version, 10, 32)
	if err != nil {
		log.Fatalf("Unable to read version %s %v\n", version, err)
	}
	log.Printf("Got file manager database version %v\n", ver)
	if ver > _DATABASE_CODE_VER {
		log.Fatalf("File manager database is too new(%v). Version is %v\n", ver, _DATABASE_CODE_VER)
	}
	if ver < _DATABASE_CODE_VER {
		if ver == 1 {
			schema := strings.Split("begin transaction;" +
			                        "create index file_states_full_path ON file_states(full_path);" + 
			                        "create index file_states_id ON file_states(id);" +
			                        "update system set value = 2 where name=\"version\";" +
			                        "commit;", ";")
			for _, sql := range schema {
				if strings.TrimSpace(sql) == "" {
					continue
				}
				err = self.conn.Exec(sql + ";")
				if err != nil {
					log.Printf("Failed to upgrade schema! %v\nSQL: %s\n", err, sql)
					os.Exit(1)
				}
			}
			ver = 2;
		}
	}
	if ver != _DATABASE_CODE_VER {
		log.Fatalf("File state manager database needs to be upgraded(%v). I'm %v\n", ver, _DATABASE_CODE_VER)
	}
}

func newFileStateManager(filename string) *fileStateManager {
	f := new(fileStateManager)
	f.db = newStateDatabase(filename)
	return f
}

//TODO - merge duplicate code here with getFileState
func (self *stateDatabase) getFileStateById(id int64) *fileState {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	s := self.getFileStateByIdStmt
	err := s.Exec(id)
	if err != nil {
		log.Fatalf("SQL statement %s execution failed with error %v", sqlGetFileStateById, err)
	}

	if !s.Next() {
		return nil
	}

	var last_modified int64
	var digest string
	var full_path string
	var last_seen int64
	var bundle_id string
	var bundle_file_id string
	var pass_off int
	var user_name string
	var rule_name string
	var auto_added_to_bundle_time int64
	err = s.Scan(&id, &last_modified, &digest, &full_path, &last_seen, &bundle_id,
		&bundle_file_id, &pass_off, &user_name, &rule_name, &auto_added_to_bundle_time)
	if err != nil {
		log.Fatalf("Scan failed with error %v", err)
	}

	var fs fileState
	fs.id = id
	fs.lastModified = last_modified
	fs.digest = digest
	fs.fullPath = full_path
	fs.lastSeen = last_seen
	if bundle_id != "" {
		i, err := strconv.Atoi(bundle_id)
		if err != nil {
			log.Fatalf("SQL query %s returned unexpected bundle_id %s, error %v", sqlGetFileStateById, bundle_id, err)
		}
		fs.bundleId = &i
	}
	if bundle_file_id != "" {
		i, err := strconv.Atoi(bundle_file_id)
		if err != nil {
			log.Fatalf("SQL query %s returned unexpected bundle_file_id %s, error %v", sqlGetFileStateById, bundle_file_id, err)
		}
		fs.bundleFileId = &i
	}
	fs.passOff = autoPassOffState(pass_off)
	fs.userName = user_name
	fs.ruleName = rule_name
	fs.autoAddedToBundleTime = auto_added_to_bundle_time

	return &fs
}

// Get a fileState from the database using the supplied user, rule, and file names.  If none exists, returns nil.
// Major errors with database cause the program to log, then exit.
func (self *stateDatabase) getFileState(userName, ruleName, fileName string) *fileState {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	s := self.getFileStateStmt

	err := s.Exec(userName, ruleName, fileName)
	if err != nil {
		log.Fatalf("SQL statement %s execution failed with error %v", sqlGetFileState, err)
	}

	if !s.Next() {
		return nil
	}

	var id int64
	var last_modified int64
	var digest string
	var full_path string
	var last_seen int64
	var bundle_id string
	var bundle_file_id string
	var pass_off int
	var user_name string
	var rule_name string
	var auto_added_to_bundle_time int64
	err = s.Scan(&id, &last_modified, &digest, &full_path, &last_seen, &bundle_id,
		&bundle_file_id, &pass_off, &user_name, &rule_name, &auto_added_to_bundle_time)
	if err != nil {
		log.Fatalf("Scan failed with error %v", err)
	}

	var fs fileState
	fs.id = id
	fs.lastModified = last_modified
	fs.digest = digest
	fs.fullPath = full_path
	fs.lastSeen = last_seen
	if bundle_id != "" {
		i, err := strconv.Atoi(bundle_id)
		if err != nil {
			log.Fatalf("SQL query %s returned unexpected bundle_id %s, error %v", sqlGetFileState, bundle_id, err)
		}
		fs.bundleId = &i
	}
	if bundle_file_id != "" {
		i, err := strconv.Atoi(bundle_file_id)
		if err != nil {
			log.Fatalf("SQL query %s returned unexpected bundle_file_id %s, error %v", sqlGetFileState, bundle_file_id, err)
		}
		fs.bundleFileId = &i
	}
	fs.passOff = autoPassOffState(pass_off)
	fs.userName = user_name
	fs.ruleName = rule_name
	fs.autoAddedToBundleTime = auto_added_to_bundle_time

	return &fs
}

func (self *stateDatabase) setFileState(fs *fileState) error {
	if fs == nil {
		return errors.New("fs is nil")
	}

	tmp := self.getFileState(fs.userName, fs.ruleName, fs.fullPath)

	self.mutex.Lock()
	defer self.mutex.Unlock()

	var sql string

	if tmp == nil { //fs is a new fileState in the database.
		//The insert statement relies on AUTOINCREMENT
		if fs.bundleId == nil || fs.bundleFileId == nil {
			sql = fmt.Sprintf(
				"INSERT INTO file_states"+
					"(id, last_modified, digest, full_path, last_seen, bundle_id, bundle_file_id, pass_off, user_name, rule_name, auto_added_to_bundle_time) "+
					"VALUES "+
					"(NULL, %d, \"%s\", \"%s\", %d, NULL, NULL, %d, \"%s\", \"%s\", %d);",
				fs.lastModified, fs.digest, fs.fullPath, fs.lastSeen, fs.passOff, fs.userName, fs.ruleName, fs.autoAddedToBundleTime)
		} else {
			sql = fmt.Sprintf(
				"INSERT INTO file_states"+
					"(id, last_modified, digest, full_path, last_seen, bundle_id, bundle_file_id, pass_off, user_name, rule_name, auto_added_to_bundle_time) "+
					"VALUES "+
					"(NULL, %d, \"%s\", \"%s\", %d, %d, %d, %d, \"%s\", \"%s\", %d);",
				fs.lastModified, fs.digest, fs.fullPath, fs.lastSeen, fs.bundleId, fs.bundleFileId, fs.passOff, fs.userName, fs.ruleName, fs.autoAddedToBundleTime)
		}
	} else { //fs is in the database, updating.
		if fs.bundleId == nil || fs.bundleFileId == nil {
			sql = fmt.Sprintf("UPDATE file_states "+
				"SET last_modified=%d, digest=\"%s\", full_path=\"%s\", last_seen=%d, pass_off=%d, user_name=\"%s\", rule_name=\"%s\", auto_added_to_bundle_time=%d "+
				"WHERE id=%d;",
				fs.lastModified, fs.digest, fs.fullPath, fs.lastSeen, fs.passOff, fs.userName, fs.ruleName, fs.autoAddedToBundleTime, fs.id)
		} else {
			sql = fmt.Sprintf("UPDATE file_states "+
				"SET last_modified=%d, digest=\"%s\", full_path=\"%s\", last_seen=%d, bundle_id=%d, bundle_file_id=%d, pass_off=%d, user_name=\"%s\", rule_name=\"%s\", auto_added_to_bundle_time=%d "+
				"WHERE id=%d;",
				fs.lastModified, fs.digest, fs.fullPath, fs.lastSeen, *fs.bundleId, *fs.bundleFileId, fs.passOff, fs.userName, fs.ruleName, fs.autoAddedToBundleTime, fs.id)
		}
	}

	common.Dprintf("%s", sql)

	return self.conn.Exec(sql)
}

//TODO - implement
//This method removes entries from file_states that no longer exist on the 
//system and have not for _STATE_RETENTION_TIME.  It should run periodically, 
//but it is not necessary to run frequently.
func (self *stateDatabase) removeDeleted() {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	//TODO - replace sqlite...
	/*log.Printf("Cleaning up orphaned files from file state database.")
	toRemove := make([]string, 0)

	for k, v := range self.states {
		exists, _ := fileExists(k)
		if now := time.Now().UnixNano(); !exists && now-v.LastSeen > _STATE_RETENTION_TIME {
			log.Printf("%s does not exist and has not been seen for %d days, removing file state from database",
				k, _STATE_RETENTION_TIME/_NANO_SECONDS_IN_DAY)
			toRemove = append(toRemove, k)
		} else if !exists {
			common.Dprintf("%s does not exist, but it is not yet old enough to remove.", k)
		} else {
			v.LastSeen = now
		}
	}

	for i := 0; i < len(toRemove); i++ {
		delete(self.states, toRemove[i])
	}*/
}

func (self *stateDatabase) setUserBundles(ub *userBundles) error {
	common.Dprintln("Entering setUserBundles")
	defer common.Dprintln("Leaving setUserBundles")

	if ub == nil {
		return errors.New("ub must not be nil")
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()

	var autoSql string
	if ub.autoSubmitBid == nil {
		autoSql = "NULL"
		ub.autoSubmitLastTouched = 0
	} else {
		autoSql = fmt.Sprintf("%d", *ub.autoSubmitBid)
		ub.autoSubmitLastTouched = time.Now().UnixNano()
	}

	var noAutoSql string
	if ub.noAutoSubmitBid == nil {
		noAutoSql = "NULL"
	} else {
		noAutoSql = fmt.Sprintf("%d", *ub.noAutoSubmitBid)
	}

	sql := fmt.Sprintf("INSERT OR REPLACE INTO user_bundles "+
		"(user, auto_submit_bid, no_auto_submit_bid, auto_submit_last_touched) "+
		"VALUES "+
		"(\"%s\", %s, %s, %d);",
		ub.user, autoSql, noAutoSql, ub.autoSubmitLastTouched)

	common.Dprintf("%s", sql)

	err := self.conn.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// Get a map of current working (i.e. in progress or yet to be submitted) bundle id's for each user.
func (self *stateDatabase) getWorkingBundles() map[string]*userBundles {
	common.Dprintln("Entering getWorkingBundles")
	defer common.Dprintln("Leaving getWorkingBundles")

	self.mutex.Lock()
	defer self.mutex.Unlock()

	sql := fmt.Sprintf("SELECT user, auto_submit_bid, no_auto_submit_bid, auto_submit_last_touched FROM user_bundles")
	s, err := self.conn.Prepare(sql)
	if err != nil {
		log.Fatalf("getWorkingBundles query preparation %s, failed with error %v", sql, err)
	}
	defer s.Finalize()

	err = s.Exec()
	if err != nil {
		log.Fatalf("getWorkingBundles query execution %s, failed with error %v", sql, err)
	}

	ret := make(map[string]*userBundles)
	var user string
	var auto_submit_bid string
	var no_auto_submit_bid string
	var auto_submit_last_touched int64
	for {
		if !s.Next() {
			break
		}
		err = s.Scan(&user, &auto_submit_bid, &no_auto_submit_bid, &auto_submit_last_touched)
		if err != nil {
			log.Fatalf("%v", err)
		}
		ub := new(userBundles)
		ub.user = user
		if auto_submit_bid != "" {
			i, err := strconv.Atoi(auto_submit_bid)
			if err != nil {
				log.Fatalf("SQL query %s returned unexpected auto_submit_bid %s, error %v", sql, auto_submit_bid, err)
			}
			ub.autoSubmitBid = &i
		}
		if no_auto_submit_bid != "" {
			i, err := strconv.Atoi(no_auto_submit_bid)
			if err != nil {
				log.Fatalf("SQL query %s returned unexpected no_bundle_file_id %s, error %v", sql, no_auto_submit_bid, err)
			}
			ub.noAutoSubmitBid = &i
		}
		ub.autoSubmitLastTouched = auto_submit_last_touched
		ret[user] = ub
	}

	return ret
}

//Gets fileStates that were in the process of being added to a bundle but for 
//whatever reason (e.g. program crash or termination) they were not completely
//added or the state change did not take affect.
func (self *stateDatabase) getFileStatesInLimbo() ([]*fileState, error) {
	//Get a list of fileState id's we need.
	self.mutex.Lock()
	sql := fmt.Sprintf("SELECT id FROM file_states WHERE pass_off=%d;", addingToBundle)
	s, err := self.conn.Prepare(sql)
	if err != nil {
		log.Fatalf("getFileStates SQL %s failed with error %v", sql, err)
	}
	defer s.Finalize()

	err = s.Exec()
	if err != nil {
		log.Fatalf("Exec %s failed, %v", sql, err)
	}

	ids := make([]int64, 0)
	for {
		if !s.Next() {
			break
		}
		var id int64
		err = s.Scan(&id)
		if err != nil {
			log.Fatalf("Scan failed %v", err)
		}
		ids = append(ids, id)
	}
	self.mutex.Unlock()

	//Using the fileState id's, get the corresponding fileStates
	states := make([]*fileState, 0)
	for _, v := range ids {
		fs := self.getFileStateById(v)
		if fs == nil {
			log.Printf("Failed to get a fileState with id %v", v)
			continue
		}
		states = append(states, fs)
	}

	return states, nil
}

//FIXME go through all this code in the whole file and make sure errors return early and unlock properly.

//Gets user/bundles that are progressing through the bundle manager
func (self *stateDatabase) getFileStatesProgressing() ([]string, []int, error) {
	self.mutex.Lock()
	sql := fmt.Sprintf("SELECT DISTINCT user_name, bundle_id FROM file_states WHERE pass_off!=%d and bundle_id IS NOT NULL;", done)
	s, err := self.conn.Prepare(sql)
	if err != nil {
		log.Fatalf("getFileStates SQL %s failed with error %v", sql, err)
	}
	defer s.Finalize()

	err = s.Exec()
	if err != nil {
		log.Fatalf("Exec %s failed, %v", sql, err)
	}

	ids := make([]int, 0)
	users := make([]string, 0)
	for {
		if !s.Next() {
			break
		}
		var id int
		var user_name string
		err = s.Scan(&user_name, &id)
		if err != nil {
			log.Fatalf("Scan failed %v", err)
		}
		ids = append(ids, id)
		users = append(users, user_name)
	}
	self.mutex.Unlock()
	return users, ids, nil
}

func (self *stateDatabase) getFileStatesByBundleFileId(bfid int) ([]*fileState, error) {
	self.mutex.Lock()

	s := self.getFileStatesByBundleFileIdStmt

	err := s.Exec(bfid)
	if err != nil {
		log.Fatalf("Exec %s failed, %v", sqlGetFileStatesByBundleFileId, err)
	}

	ids := make([]int64, 0)
	for {
		if !s.Next() {
			break
		}
		var id int64
		err = s.Scan(&id)
		if err != nil {
			log.Fatalf("Scan failed %v", err)
		}
		ids = append(ids, id)
	}

	self.mutex.Unlock()

	//Using the fileState id's, get the corresponding fileStates
	states := make([]*fileState, 0)
	for _, v := range ids {
		fs := self.getFileStateById(v)
		if fs == nil {
			log.Printf("Failed to get a fileState with id %v", v)
			continue
		}
		states = append(states, fs)
	}

	return states, nil
}

func fileExists(name string) (bool, error) {
	fi, err := os.Stat(name)
	if err == nil && !fi.IsDir() {
		return true, nil
	}

	if fi.IsDir() {
		return false, nil
	}

	return false, err
}

func (self *fileStateManager) setFileState(fs *fileState) error {
	return self.db.setFileState(fs)
}

func (self *fileStateManager) setFileStateCreate(userName string, ruleName string, fullPath string,
	bundleId *int, bundleFileId *int, state autoPassOffState) error {
	fi, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	fs := &fileState{lastModified: fi.ModTime().UnixNano(),
		fullPath:              fullPath,
		lastSeen:              time.Now().UnixNano(),
		bundleId:              bundleId,
		bundleFileId:          bundleFileId,
		passOff:               state,
		userName:              userName,
		ruleName:              ruleName,
		autoAddedToBundleTime: 0}

	return self.db.setFileState(fs)
}

// Get fileState for arguments.  Create and return one if none exists.
func (self *fileStateManager) getFileState(user, rulename string, fullpath string) (*fileState, error) {
	fs := self.getFileStateNoCreate(user, rulename, fullpath)
	if fs == nil {
		//This block creates the fileState
		err := self.setFileStateCreate(user, rulename, fullpath, nil, nil, notSeenBefore)
		if err != nil {
			msg := fmt.Sprintf("setFileState failed with error message %v", err)
			return nil, errors.New(msg)
		}
		// This block gets the fileState that was just created.  This accomplishes two goals.
		//	1. We know it is in the database.
		//	2. It comes back with the correct id that was auto-generated by the database.
		fs = self.getFileStateNoCreate(user, rulename, fullpath)
		if fs == nil {
			msg := "A new fileState was created, set, and retrieved. " +
				"The retrieve (getFileState) failed, which was unexpected."
			return nil, errors.New(msg)
		}
	}
	return fs, nil
}

// Get fileState for arguments.  Returns nil if not found with an error message.
func (self *fileStateManager) getFileStateNoCreate(user string, rulename string, fullpath string) *fileState {
	return self.db.getFileState(user, rulename, fullpath)
}

func (self *fileStateManager) getFileStatesInLimbo() ([]*fileState, error) {
	return self.db.getFileStatesInLimbo()
}
