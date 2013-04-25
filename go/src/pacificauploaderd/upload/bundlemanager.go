package upload

import (
	"os"
	"fmt"
	"log"
	"sync"
	"time"
	"errors"
	"sqlite"
	"strconv"
	"strings"
	"net/http"
	"encoding/json"
	"path/filepath"
	"pacificauploaderd/web"
	"pacificauploaderd/common"
)

const (
	DBCODEVER = 2
)

var (
	ErrBundleNotFound error = errors.New("BundleNotFound")
)

type BundleManagerStateChangeWakeFunc func(user string, bundle_id int, state BundleState)

type BundleManager struct {
	conn *sqlite.Conn
	mutex sync.Mutex
	shutdown bool
	stateChangeFuncs map[BundleState][]BundleManagerStateChangeWakeFunc
}

// Represents a bundle (tar file).  Use this for accessing higher level methods for
// manipulation of a bundle (tar file)
type BundleMD struct {
	id int
	bm *BundleManager
	user string
}

// Represents a file inside a BundleMD.
type BundleFileMD struct {
	bundle *BundleMD
	id int
	conn *sqlite.Conn
}

func BundleManagerNew() *BundleManager {
	self := &BundleManager{shutdown: false}
	self.Init()
	return self
}

func (self *BundleMD) Submit() error {
	return self.bm.BundleStateSet(self.user, self.id, BundleState_ToBundle)
}

func (self *BundleMD) WatchAdd(watcher string) error {
	return self.bm.bundleWatchAddNolock(self.user, watcher, self.id, false)
}

func (self *BundleMD) Delete(watcher string) error {
	return self.bm.BundleDelete(self.user, watcher, self.id)
}

func (self *BundleMD) FlaggedToDelete() (bool, error) {
	return self.bm.bundleFlaggedToDelete(self.user, self.id)
}


func (self *BundleMD) FileIdsGet() ([]int, error) {
	return self.bm.BundleFileIdsGet(self.user, self.id)
}

func (self *BundleMD) FilesGet() ([]*BundleFileMD, error) {
	retval := []*BundleFileMD{}
	list, err := self.bm.BundleFileIdsGet(self.user, self.id)
	if err != nil {
		return retval, err
	}
	for _, id := range list {
		bf, err := self.FileGet(id)
		if err != nil {
			return retval, err
		}
		retval = append(retval, bf)
	}
	return retval, nil
}

func (self *BundleMD) FileServiceGet() (string, error) {
	return self.bm.BundleStringGet(self.user, self.id, "file_service")
}

func (self *BundleMD) ErrorGet() (string, error) {
	return self.bm.BundleStringGet(self.user, self.id, "error")
}

func (self *BundleMD) TransactionGet() (string, error) {
	return self.bm.BundleStringGet(self.user, self.id, "trans_id")
}

func (self *BundleMD) AvailableGet() (bool, error) {
	return self.bm.BundleBoolGet(self.user, self.id, "available")
}

func (self *BundleMD) StateGet() (BundleState, error) {
	return self.bm.BundleStateGet(self.user, self.id)
}

func (self *BundleMD) StateSet(state BundleState) error {
	return self.bm.BundleStateSet(self.user, self.id, state)
}

func (self *BundleMD) FileGet(id int) (*BundleFileMD, error) {
	return self.bm.BundleFileGet(self, "", -1, id)
}

func (self *BundleMD) FileAdd(pacifica_filename string, local_filename string, commit bool) (*BundleFileMD, error) {
	var conn *sqlite.Conn
	if commit == false {
		var err error
		conn, err = self.bm.connGet()
		if err != nil {
			return nil, err
		}
		err = conn.Exec("begin transaction")
		if err != nil {
			return nil, err
		}
	} else {
		conn = self.bm.conn
	}
	id, err := self.bm.bundleFileIdAdd(conn, self.user, self.id, pacifica_filename, local_filename)
	if err != nil {
		if commit == false {
			conn.Close()
		}
		return nil, err
	}
	return self.bm.bundleFileGet(conn, self, "", -1, id)
}

func (self *BundleMD) StatusUrlGet() (string, error) {
	return self.bm.BundleStringGet(self.user, self.id, "status_url")
}

func (self *BundleMD) statusUrlSet(url string) error {
	return self.bm.bundleStringSet(self.user, self.id, "status_url", url)
}

func (self *BundleMD) BundleLocationGet() (string, error) {
	return self.bm.BundleStringGet(self.user, self.id, "bundle_location")
}

func (self *BundleMD) bundleLocationSet(location string) error {
	return self.bm.bundleStringSet(self.user, self.id, "bundle_location", location)
}

func (self *BundleMD) IdGet() int {
	return self.id
}

//Returns the number of BundleFileMD this BundleMD have been added to it.  If error is returned,
//int64 will be invalid.
func (self *BundleMD) BundleFileCountGet() (int64, error) {
	self.bm.mutex.Lock()
	if self.bm.shutdown {
		return 0, errors.New("BundleManager is shutdown.")
	}
	defer self.bm.mutex.Unlock()
	sql := "SELECT COUNT(bundle_id) FROM files WHERE bundle_id=?;"
	s, err := self.bm.conn.Prepare(sql)
	if err != nil {
		log.Printf("%v %v\n", sql, err)
		return 0, err
	}
	defer s.Finalize()
	err = s.Exec(self.id)
	if err != nil {
		log.Printf("%v %v\n", sql, err)
		return 0, err
	}
	if !s.Next() {
		return 0, errors.New("Failed to get count for bundle files")
	}
	var retval int64
	err = s.Scan(&retval)
	if err != nil {
		log.Printf("%v\n", err)
		return 0, err
	}
	return retval, nil
}

func (self *BundleFileMD) IdGet() int {
	return self.id
}

func (self *BundleFileMD) Commit() error {
	if self.conn == nil {
		return nil
	}
	err := self.conn.Exec("commit")
	self.conn.Close()
	self.conn = nil
	return err
}

func (self *BundleFileMD) LocalFilenameGet() (string, error) {
	return self.bundle.bm.BundleFileStringGet(self.bundle.user, self.bundle.id, self.id, "local_filename")
}

func (self *BundleFileMD) LocalFilenameSet(value string) error {
	return self.bundle.bm.BundleFileStringSet(self.bundle.user, self.bundle.id, self.id, "local_filename", value)
}

func (self *BundleFileMD) PacificaFilenameGet() (string, error) {
	return self.bundle.bm.BundleFileStringGet(self.bundle.user, self.bundle.id, self.id, "myemsl_filename")
}

func (self *BundleFileMD) PacificaFilenameSet(value string) error {
	return self.bundle.bm.BundleFileStringSet(self.bundle.user, self.bundle.id, self.id, "myemsl_filename", value)
}

func (self *BundleFileMD) Sha1Get() (string, error) {
	return self.bundle.bm.BundleFileStringGet(self.bundle.user, self.bundle.id, self.id, "sha1")
}

func (self *BundleFileMD) Sha1Set(value string) error {
	return self.bundle.bm.BundleFileStringSet(self.bundle.user, self.bundle.id, self.id, "sha1", value)
}

func (self *BundleFileMD) MtimeGet() (*time.Time, error) {
	tstr, err := self.bundle.bm.BundleFileStringGet(self.bundle.user, self.bundle.id, self.id, "mtime")
	if err != nil {
		return nil, err
	}
	if tstr == "" {
		return nil, nil
	}
	t := time.Time{};
	err = t.UnmarshalJSON([]byte(tstr))
	return &t, err
}

func (self *BundleFileMD) MtimeSet(value time.Time) error {
	bytes, err := value.MarshalJSON()
	if err != nil {
		return err
	}
	return self.bundle.bm.BundleFileStringSet(self.bundle.user, self.bundle.id, self.id, "mtime", string(bytes))
}

func (self *BundleFileMD) GroupsGet() ([][2]string, error) {
	return self.bundle.bm.BundleFileGroupsGet(self.bundle.user, self.bundle.id, self.id)
}

func (self *BundleFileMD) GroupsSet(groups [][2]string) error {
	return self.bundle.bm.BundleFileGroupsSet(self.conn, self.bundle.user, self.bundle.id, self.id, groups)
}

func (self *BundleFileMD) DisableOnErrorGet() (bool, error) {
	return self.bundle.bm.BundleFileBoolGet(self.bundle.user, self.bundle.id, self.id, "disable_on_error")
}

func (self *BundleFileMD) DisableOnErrorSet(state bool) error {
	return self.bundle.bm.BundleFileBoolSet(self.conn, self.bundle.user, self.bundle.id, self.id, "disable_on_error", state)
}

func (self *BundleFileMD) DisableOnErrorMsgGet() (string, error) {
	return self.bundle.bm.BundleFileStringGet(self.bundle.user, self.bundle.id, self.id, "disable_on_error_msg")
}

func (self *BundleManager) connGet() (*sqlite.Conn, error) {
	conn, err := sqlite.Open(filepath.Join(common.StateDir, "bundlemanager.sdb"))
	if err != nil {
		log.Printf("connGet-Open: %v\n", err)
		return nil, err
	}
	conn.BusyTimeout(1000 * 60 * 10) //Ten min.
	err = conn.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Printf("connGet-foreign_keys: %v\n", err)
		return nil, err
	}
	return conn, nil
}

func (self *BundleManager) Init() {
	var err error
	self.stateChangeFuncs = make(map[BundleState][]BundleManagerStateChangeWakeFunc)
	self.conn, err = self.connGet()
	if err != nil {
		os.Exit(1)
	}
	s, err := self.conn.Prepare("select value from system where name = \"version\";")
	if err != nil && err.Error() == "SQL error or missing database: no such table: system" {
		err = self.conn.Exec("create table system(name string primary key, value string);")
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(1)
		}
		schema := strings.Split(`
create table bundles(
	id integer primary key,
	user string not null,
	state integer not null,
	status_url string,
	trans_id string,
	bundle_location string,
	file_service string,
	available bool not null default false,
	error string
);
create table files(
	id integer primary key,
	bundle_id integer not null,
	sha1 string,
	myemsl_filename string not null,
	local_filename string not null,
	mtime string,
	disable_on_error bool not null default false,
	disable_on_error_msg string,
	foreign key(bundle_id) references bundles(id)
);
create table groups(
	file_id integer not null,
	type string not null,
	name string not null,
	foreign key(file_id) references files(id)
);
create table bundle_watches(
	bundle_id integer not null,
	name string not null,
	deleted bool not null default false,
	foreign key(bundle_id) references bundles(id)
);
create unique index filesi1 on files(bundle_id, myemsl_filename);
create unique index groupsi1 on groups(file_id, type, name);
create unique index bwi1 on bundle_watches(bundle_id, name);
create index bundles_id ON bundles(id);
create index files_id ON files(id);
`, ";")
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
		err = self.conn.Exec("insert into system(name, value) values(\"version\", ?);", int(DBCODEVER))
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(1)
		}
		s, err = self.conn.Prepare("select value from system where name = \"version\";")
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(1)
		}
	}
	defer s.Finalize()
	err = s.Exec()
	if err != nil {
		log.Printf("%v\n", err)
		os.Exit(1)
	}
	version := ""
	for {
		if !s.Next() {
			break;
		}
		var value string
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(1)
		}
		version = value
	}
	if version == "" {
		log.Printf("Failed to get a version from database.\n")
		os.Exit(1)
	}
	if version == "" {
		log.Printf("Failed to get a version from database.\n")
		os.Exit(1)
	}
	ver, err := strconv.ParseInt(version, 10, 32)
	if err != nil {
		log.Printf("Unable to read version %s %v\n", version, err)
		os.Exit(1)
	}
	log.Printf("Got Bundle Manager Database Version %v\n", ver)
	if ver > DBCODEVER {
		log.Printf("Bundle Manager Database is too new(%v). I'm %v\n", ver, DBCODEVER)
		os.Exit(1)

	}
	if ver < DBCODEVER {
		if ver == 1 {
			schema := strings.Split("begin transaction;" +
			                        "create index bundles_id ON bundles(id);" + 
			                        "create index files_id ON files(id);" +
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
	if ver != DBCODEVER {
		log.Printf("Bundle Manager Database needs to be upgraded(%v). I'm %v\n", ver, DBCODEVER)
		os.Exit(1)

	}
}

func (self *BundleManager) Stop() {
	if self.shutdown {
		return
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.shutdown = true
	self.conn.Close()
}

func (self *BundleManager) BundleIdsGet(user string, watcher string) ([]int, error) {
	self.mutex.Lock()
	if self.shutdown {
		return nil, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	retval := []int{}
	var sql string
	if watcher == "" {
		sql = "select id from bundles where user = ?;"
	} else {
		sql = "select id from bundles left outer join bundle_watches on bundles.id = bundle_watches.bundle_id and bundle_watches.name = ? where (bundle_watches.bundle_id is null or bundle_watches.deleted = 0) and user = ? group by id;" 
	}
	s, err := self.conn.Prepare(sql)
	if err != nil {
		log.Printf("%v %v\n", sql, err)
		return nil, err
	}
	defer s.Finalize()
	if watcher == "" {
		err = s.Exec(user)
	} else {
		err = s.Exec(watcher, user)
	}
	if err != nil {
		log.Printf("%v %v\n", sql, err)
		return nil, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return nil, err
		}
		retval = append(retval, value)
	}
	return retval, nil
}

func (self *BundleManager) bundleIdsForStateGet(state BundleState) (map[string][]int, error) {
	self.mutex.Lock()
	if self.shutdown {
		return nil, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	retval := make(map[string][]int, 0)
	s, err := self.conn.Prepare("select user, id from bundles where state = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	defer s.Finalize()
	err = s.Exec(int(state))
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	for {
		if !s.Next() {
			break;
		}
		var user string
		var value int
		err = s.Scan(&user, &value)
		if err != nil {
			log.Printf("%v\n", err)
			return nil, err
		}
		if retval[user] == nil {
			retval[user] = []int{value}
		} else {
			retval[user] = append(retval[user], value)
		}
	}
	return retval, nil
}

func (self *BundleManager) bundleUsersForState(state BundleState, cm *CredsManager) ([]string, error) {
	self.mutex.Lock()
	if self.shutdown {
		return nil, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	retval := make([]string, 0)
	s, err := self.conn.Prepare("select user from bundles where state = ? group by user;")
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	defer s.Finalize()
	err = s.Exec(int(state))
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	for {
		if !s.Next() {
			break;
		}
		var user string
		err = s.Scan(&user)
		if err != nil {
			log.Printf("%v\n", err)
			return nil, err
		}
//FIXME make this configurable
		if cm != nil && cm.userCreds[user] != nil && (cm.outage[user] == nil || cm.outage[user].Add(time.Duration(10) * time.Minute).Before(time.Now())) {
			retval = append(retval, user)
		}
	}
	return retval, nil
}

func (self *BundleManager) bundleIdsForStateCount(state BundleState) (int, error) {
	self.mutex.Lock()
	if self.shutdown {
		return -1, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	retval := 0
	s, err := self.conn.Prepare("select count(id) from bundles where state = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return -1, err
	}
	defer s.Finalize()
	err = s.Exec(int(state))
	if err != nil {
		log.Printf("%v\n", err)
		return -1, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return -1, err
		}
		retval = value
	}
	return retval, nil
}

func (self *BundleManager) BundleIdAdd(user string) (int, error) {
	self.mutex.Lock()
	if self.shutdown {
		return -1, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	err := self.conn.Exec("insert into bundles(user, state) values(?, ?);", user, int(BundleState_Unsubmitted))
	if err != nil {
		log.Printf("%v\n", err)
		return -1, err
	}
	s, err := self.conn.Prepare("select last_insert_rowid();")
	if err != nil {
		log.Printf("%v\n", err)
		return -1, err
	}
	defer s.Finalize()
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return -1, err
		}
		return value, nil
	}
	return -1, errors.New("Unknown error.")
}

func (self *BundleManager) BundleGet(user string, id int) (*BundleMD, error) {
	self.mutex.Lock()
	if self.shutdown {
		return nil, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	s, err := self.conn.Prepare("select 1 from bundles where user = ? and id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	defer s.Finalize()
	err = s.Exec(user, id)
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return nil, err
		}
		if value == 1 {
			return &BundleMD{id: id, bm: self, user: user}, nil
		}
	}
	return nil, ErrBundleNotFound
}

func (self *BundleManager) BundleAdd(user string) (*BundleMD, error) {
	id, err := self.BundleIdAdd(user)
	if err != nil {
		return nil, err
	}
	return self.BundleGet(user, id)
}

func (self *BundleManager) BundleFileIdsGet(user string, bundle_id int) ([]int, error) {
	self.mutex.Lock()
	if self.shutdown {
		return nil, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	retval := []int{}
	s, err := self.conn.Prepare("select files.id from files, bundles where bundles.user = ? and bundles.id = ? and bundles.id = files.bundle_id;")
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	defer s.Finalize()
	err = s.Exec(user, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return nil, err
		}
		retval = append(retval, value)
	}
	return retval, nil
}

func (self *BundleManager) BundleFileGet(bundle *BundleMD, user string, bundle_id int, file_id int) (*BundleFileMD, error) {
	return self.bundleFileGet(self.conn, bundle, user, bundle_id, file_id)
}

func (self *BundleManager) bundleFileGet(conn *sqlite.Conn, bundle *BundleMD, user string, bundle_id int, file_id int) (*BundleFileMD, error) {
	self.mutex.Lock()
	if self.shutdown {
		return nil, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	if bundle != nil {
		bundle_id = bundle.id
		user = bundle.user
	} else {
		bundle = &BundleMD{id: bundle_id, user: user, bm: self}
	}
	s, err := conn.Prepare("select 1 from files, bundles where bundles.user = ? and bundles.id = ? and files.bundle_id = bundles.id and files.id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	defer s.Finalize()
	err = s.Exec(user, bundle_id, file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return nil, err
		}
		if value == 1 {
			if conn == self.conn {
				conn = nil
			}
			return &BundleFileMD{id: file_id, bundle: bundle, conn: conn}, nil
		}
	}
	return nil, errors.New("File Id " + strconv.Itoa(file_id) + " not found.")
}

func (self *BundleManager) BundleFileIdAdd(user string, bundle_id int, pacifica_filename string, local_filename string) (int, error) {
	return self.bundleFileIdAdd(self.conn, user, bundle_id, pacifica_filename, local_filename)
}

func (self *BundleManager) bundleFileIdAdd(conn *sqlite.Conn, user string, bundle_id int, pacifica_filename string, local_filename string) (int, error) {
	self.mutex.Lock()
	if self.shutdown {
		return -1, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	sql := `
insert into files(bundle_id, local_filename, myemsl_filename) values(
	(select case when (state == ?) then ? else null end from bundles where user = ? and id = ?),
	?,
	?
);`
	err := conn.Exec(sql, int(BundleState_Unsubmitted), bundle_id, user, bundle_id, local_filename, pacifica_filename)
	if err != nil {
		old_state, terr := self.bundleStateGetNoLock(conn, user, bundle_id)
		if terr != nil {
			log.Printf("%v\n", terr)
			return -1, err
		}
		if old_state != BundleState_Unsubmitted {
			return -1, errors.New("Can't add files to submitted bundle.")
		}
		s, terr := conn.Prepare("select id, local_filename from files where bundle_id = ? and myemsl_filename = ?;")
		if terr != nil {
			log.Printf("%v\n", terr)
			return -1, err
		}
		defer s.Finalize()
		terr = s.Exec(bundle_id, pacifica_filename)
		if terr != nil {
			log.Printf("%v\n", terr)
			return -1, err
		}
		for {
			if !s.Next() {
				break;
			}
			var tid int
			var tlocal_filename string
			terr = s.Scan(&tid, &tlocal_filename)
			if terr != nil {
				log.Printf("%v\n", terr)
				return -1, err
			}
			if(tlocal_filename == local_filename) {
				return -1, errors.New(fmt.Sprintf("The file with the specified Server filename (%s) already exists in the bundle (%d)!", pacifica_filename, bundle_id))
			} else {
				return -1, errors.New(fmt.Sprintf("The file with the specified Server filename (%s) already exists in the bundle (%d) but with a different local filename (%s != %s)!", pacifica_filename, bundle_id, local_filename, tlocal_filename))
			}
		}
		log.Printf("%v\n", err)
		log.Printf("SQL: %s, %d, %d, %s, %d, %s, %s\n", sql, int(BundleState_Unsubmitted), bundle_id, user, bundle_id, local_filename, pacifica_filename)
		return -1, err
	}
	s, err := conn.Prepare("select last_insert_rowid();")
	if err != nil {
		log.Printf("%v\n", err)
		return -1, err
	}
	defer s.Finalize()
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return -1, err
		}
		return value, nil
	}
	return -1, errors.New("Unknown error.")
}

func (self *BundleManager) BundleFileStringGet(user string, bundle_id int, file_id int, column string) (string, error) {
	self.mutex.Lock()
	if self.shutdown {
		return "", errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	s, err := self.conn.Prepare("select " + column + " from files, bundles where bundles.user = ? and bundles.id = files.bundle_id and files.id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return "", err
	}
	defer s.Finalize()
	err = s.Exec(user, file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return "", err
	}
	for {
		if !s.Next() {
			break;
		}
		var value string
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return "", err
		}
		return value, nil
	}
	return "", errors.New("Unknown error.")
}

func (self *BundleManager) BundleFileStringSet(user string, bundle_id int, file_id int, column string, value string) error {
	self.mutex.Lock()
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	if column != "sha1" && column != "mtime" && column != "disable_on_error_msg" {
		old_state, err := self.bundleStateGetNoLock(nil, user, bundle_id)
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		if old_state != BundleState_Unsubmitted {
			return errors.New("Can't set string on submitted bundle.")
		}
	}
//FIXME check for bundle->file.
	err := self.conn.Exec("update files set " + column + " = ? where id = ?;", value, file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	return nil
}

func (self *BundleManager) BundleFileBoolGet(user string, bundle_id int, file_id int, column string) (bool, error) {
	self.mutex.Lock()
	if self.shutdown {
		return false, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	s, err := self.conn.Prepare("select " + column + " from files, bundles where bundles.user = ? and bundles.id = files.bundle_id and files.id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	defer s.Finalize()
	err = s.Exec(user, file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value bool
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return false, err
		}
		return value, nil
	}
	return false, errors.New("Unknown error.")
}

func (self *BundleManager) BundleFileBoolSet(conn *sqlite.Conn, user string, bundle_id int, file_id int, column string, value bool) error {
	self.mutex.Lock()
	if conn == nil {
		conn = self.conn
	}
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	if column != "disable_on_error" {
		old_state, err := self.bundleStateGetNoLock(conn, user, bundle_id)
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		if old_state != BundleState_Unsubmitted {
			return errors.New("Can't set string on submitted bundle.")
		}
	}
//FIXME check for bundle->file.
	err := conn.Exec("update files set " + column + " = ? where id = ?;", value, file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	return nil
}

func (self *BundleManager) BundleStringGet(user string, bundle_id int, column string) (string, error) {
	self.mutex.Lock()
	if self.shutdown {
		return "", errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	return self.bundleStringGetNolock(user, bundle_id, column)
}

func (self *BundleManager) bundleStringGetNolock(user string, bundle_id int, column string) (string, error) {
	s, err := self.conn.Prepare("select " + column + " from bundles where bundles.user = ? and bundles.id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return "", err
	}
	defer s.Finalize()
	err = s.Exec(user, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return "", err
	}
	for {
		if !s.Next() {
			break;
		}
		var value string
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return "", err
		}
		return value, nil
	}
	return "", errors.New("Unknown error.")
}

func (self *BundleManager) bundleStringSet(user string, bundle_id int, column string, value string) error {
	self.mutex.Lock()
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	if column != "status_url" && column != "bundle_location" && column != "error" && column != "file_service" && column != "trans_id" {
		old_state, err := self.bundleStateGetNoLock(nil, user, bundle_id)
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		if old_state != BundleState_Unsubmitted {
			return errors.New("Can't set string on submitted bundle.")
		}
	}
	err := self.conn.Exec("update bundles set " + column + " = ? where id = ? and user = ?;", value, bundle_id, user)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	return nil
}

func (self *BundleManager) BundleBoolGet(user string, bundle_id int, column string) (bool, error) {
	self.mutex.Lock()
	if self.shutdown {
		return false, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	s, err := self.conn.Prepare("select " + column + " from bundles where bundles.user = ? and bundles.id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	defer s.Finalize()
	err = s.Exec(user, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value bool
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return false, err
		}
		return value, nil
	}
	return false, errors.New("Unknown error.")
}

func (self *BundleManager) bundleBoolSet(user string, bundle_id int, column string, value bool) error {
	self.mutex.Lock()
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	if column != "available" {
		old_state, err := self.bundleStateGetNoLock(nil, user, bundle_id)
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		if old_state != BundleState_Unsubmitted {
			return errors.New("Can't set string on submitted bundle.")
		}
	}
	err := self.conn.Exec("update bundles set " + column + " = ? where id = ? and user = ?;", value, bundle_id, user)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	return nil
}

func (self *BundleManager) BundleStateChangeWatchSet(state BundleState, bmscwf BundleManagerStateChangeWakeFunc) error {
	self.mutex.Lock()
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	if self.stateChangeFuncs[state] == nil {
		self.stateChangeFuncs[state] = []BundleManagerStateChangeWakeFunc{}
	}
	self.stateChangeFuncs[state] = append(self.stateChangeFuncs[state], bmscwf)
	return nil
}

func (self *BundleManager) BundleStateGet(user string, bundle_id int) (BundleState, error) {
	self.mutex.Lock()
	if self.shutdown {
		return BundleState_Error, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	return self.bundleStateGetNoLock(nil, user, bundle_id)
}

func (self *BundleManager) bundleStateGetNoLock(conn *sqlite.Conn, user string, bundle_id int) (BundleState, error) {
	if conn == nil {
		conn = self.conn
	}
	s, err := conn.Prepare("select state from bundles where bundles.user = ? and bundles.id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return BundleState_Error, err
	}
	defer s.Finalize()
	err = s.Exec(user, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return BundleState_Error, err
	}
	for {
		if !s.Next() {
			break;
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return BundleState_Error, err
		}
		return BundleState(value), nil
	}
	return BundleState_Error, errors.New("Unknown error.")
}

func (self *BundleManager) BundleStateSet(user string, bundle_id int, state BundleState) error {
	self.mutex.Lock()
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	err := self.bundleStateSetNolock(user, bundle_id, state)
	self.mutex.Unlock()
	if self.stateChangeFuncs[state] != nil {
		for _, f := range self.stateChangeFuncs[state] {
			f(user, bundle_id, state)
		}
	}
	return err
}


//WARNING. Don't forget to call callbacks as needed.
func (self *BundleManager) bundleStateSetNolock(user string, bundle_id int, state BundleState) error {
	old_state, err := self.bundleStateGetNoLock(nil, user, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	if !BundleStateTransitionOk(old_state, state) {
		return errors.New(fmt.Sprintf("Transition from %v to %v is not allowed.", int(old_state), int(state)))
	}
	err = self.conn.Exec("update bundles set state = ? where user = ? and id = ?;", int(state), user, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	return nil
}

func (self *BundleManager) BundleFileGroupsGet(user string, bundle_id int, file_id int) ([][2]string, error) {
	self.mutex.Lock()
	if self.shutdown {
		return nil, errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	retval := make([][2]string, 0)
	s, err := self.conn.Prepare("select type, name from groups, files, bundles where bundles.user = ? and bundles.id = ? and bundles.id = files.bundle_id and files.id = ? and files.id = groups.file_id;")
	if err != nil {
		log.Printf("%v\n", err)
		return retval, err
	}
	defer s.Finalize()
	err = s.Exec(user, bundle_id, file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return retval, err
	}
	for {
		if !s.Next() {
			break
		}
		var t string
		var n string
		err = s.Scan(&t, &n)
		if err != nil {
			log.Printf("%v\n", err)
			return retval, err
		}
		retval = append(retval, [2]string{t, n})
	}
	return retval, nil
}

func (self *BundleManager) bundleFileValidateNolock(conn *sqlite.Conn, user string, bundle_id int, file_id int) error {
	if conn == nil {
		conn = self.conn
	}
//FIXME better way to encode username and column?
	s, err := conn.Prepare("select 1 from files, bundles where bundles.user = ? and bundles.id = ? and bundles.id = files.bundle_id and files.id = ?;")
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	defer s.Finalize()
	err = s.Exec(user, bundle_id, file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	for {
		if !s.Next() {
			break
		}
		var value int
		err = s.Scan(&value)
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		if value == 1 {
			return nil
		}
	}
	return errors.New("The specified file is not owned by the user and bundle specified.")
}

func (self *BundleManager) BundleFileGroupsSet(conn *sqlite.Conn, user string, bundle_id int, file_id int, groups [][2]string) error {
	self.mutex.Lock()
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	state, err := self.bundleStateGetNoLock(conn, user, bundle_id)
	if err != nil {
		return err
	}
	if state != BundleState_Unsubmitted {
		return errors.New("The bundle specified is not editable.")
	}
	err = self.bundleFileValidateNolock(conn, user, bundle_id, file_id)
	if err != nil {
		return nil
	}
	err = conn.Exec("delete from groups where file_id = ?;", file_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	for _, item := range groups {
		err = conn.Exec("insert into groups(file_id, type, name) values(?, ?, ?);", file_id, item[0], item[1])
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
	}
	return nil
}

func (self *BundleManager) BundleUserState(user string) string {
	self.mutex.Lock()
	if self.shutdown {
		return "error"
	}
	defer self.mutex.Unlock()
	s, err := self.conn.Prepare("select state from bundles where bundles.user = ? group by state;")
	if err != nil {
		log.Printf("%v\n", err)
		return "error"
	}
	defer s.Finalize()
	err = s.Exec(user)
	if err != nil {
		log.Printf("%v\n", err)
		return "error"
	}
	retval := "idle"
	for {
		if !s.Next() {
			break;
		}
		var tvalue int
		err = s.Scan(&tvalue)
		if err != nil {
			log.Printf("%v\n", err)
			return "error"
		}
		value := BundleState(tvalue)
		if value == BundleState_Error {
			return "error"
		}
		if value == BundleState_ToUpload {
			retval = "uploading"
		}
		if(value == BundleState_ToBundle || value == BundleState_Submitted) && retval != "uploading" {
			retval = "processing"
		}
	}
	return retval
}

func (self *BundleManager) bundleWatchAddNolock(user string, watcher string, bundle_id int, deleted bool) error {
	if watcher == "" {
		return nil
	}
	_, err := self.bundleStateGetNoLock(nil, user, bundle_id)
	if err != nil {
		log.Printf("Failed to get bundle state. %v %v\n", user, bundle_id)
		return err
	}
	err = self.conn.Exec("insert into bundle_watches(bundle_id, deleted, name) values(?, ?, ?);", bundle_id, deleted, watcher)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	return nil
}

func (self *BundleManager) bundleWatchDeleteNolock(user string, watcher string, bundle_id int) error {
	if watcher == "" {
		return nil
	}
	_, err := self.bundleStateGetNoLock(nil, user, bundle_id)
	if err != nil {
		log.Printf("Failed to get bundle state. %v %v\n", user, bundle_id)
		return err
	}
	err = self.conn.Exec("update bundle_watches set deleted = 1 where bundle_id = ? and name = ?;", bundle_id, watcher)
	if err != nil {
		err = self.bundleWatchAddNolock(user, watcher, bundle_id, true)
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
	}
	return nil
}

func (self *BundleManager) bundleAllWatchesClearedNolock(user string, bundle_id int) (bool, error) {
	_, err := self.bundleStateGetNoLock(nil, user, bundle_id)
	if err != nil {
		log.Printf("Failed to get bundle state. %v %v\n", user, bundle_id)
		return false, err
	}
	s, err := self.conn.Prepare("select count(deleted) from bundle_watches where bundle_id = ? and deleted = 0 group by deleted;")
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	defer s.Finalize()
	err = s.Exec(bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	retval := true
	for {
		if !s.Next() {
			break;
		}
		var tvalue int
		err = s.Scan(&tvalue)
		if err != nil {
			log.Printf("%v\n", err)
			return false, err
		}
		if tvalue > 0 {
			retval = false
		}
	}
	return retval, nil
}

func (self *BundleManager) bundleFlaggedToDelete(user string, bundle_id int) (bool, error) {
	self.mutex.Lock()
	if self.shutdown {
		return false, errors.New("error")
	}
	defer self.mutex.Unlock()
	_, err := self.bundleStateGetNoLock(nil, user, bundle_id)
	if err != nil {
		log.Printf("Failed to get bundle state. %v %v\n", user, bundle_id)
		return false, err
	}
	s, err := self.conn.Prepare("select count(deleted) as count from bundle_watches where bundle_id = ? and deleted != 0 group by deleted;")
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	defer s.Finalize()
	err = s.Exec(bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return false, err
	}
	retval := false
	for {
		if !s.Next() {
			break;
		}
		var tvalue int
		err = s.Scan(&tvalue)
		if err != nil {
			log.Printf("%v\n", err)
			return false, err
		}
		if tvalue != 0 {
			retval = true
		}
	}
	return retval, nil
}

func (self *BundleManager) BundleDelete(user string, watcher string, bundle_id int) error {
	self.mutex.Lock()
	if self.shutdown {
		return errors.New("BundleManager is shutdown.")
	}
	defer self.mutex.Unlock()
	state, err := self.bundleStateGetNoLock(nil, user, bundle_id)
	if err != nil {
		log.Printf("Failed to get bundle state. %v %v\n", user, bundle_id)
		return err
	}
	if BundleStateActionOk(state, BundleAction_Deletable) == false {
		log.Printf("Bundle state bad while removing: %v. %v\n", bundle_id, state)
		return errors.New("The bundle specified is not deletable.")
	}
	self.bundleWatchAddNolock(user, watcher, bundle_id, false)
	err = self.bundleWatchDeleteNolock(user, watcher, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	clear, err := self.bundleAllWatchesClearedNolock(user, bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	if clear == false {
		return nil
	}
	bundle_file, err := self.bundleStringGetNolock(user, bundle_id, "bundle_location")
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	if bundle_file != "" {
		err = os.Remove(bundle_file)
		if err != nil && os.IsNotExist(err) == false {
			log.Printf("Error removing: %v. %T %#v %#v\n", bundle_file, err, err, os.ErrNotExist)
			return err
		}
	}
	err = self.conn.Exec("delete from groups where file_id in (select id from files where bundle_id = ?);", bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	err = self.conn.Exec("delete from files where bundle_id = ?;", bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	err = self.conn.Exec("delete from bundle_watches where bundle_id = ?;", bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	err = self.conn.Exec("delete from bundles where id = ?;", bundle_id)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	return nil
}

func panic_err(err error) {
	if err != nil {
		panic(err)
	}
}

func bundleIdsHandle(bm *BundleManager, w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(w, "{\"Error\":\"%v\"}", err)
		}
	} ()
        if web.AuthCheck(w, req) {
		user := web.AuthUser(req)
		if req.Method == "GET" && req.RequestURI == "/bundle/json/" {
//FIXME allow watcher to be specified by the user of the service.
			list, err := bm.BundleIdsGet(user, "web")
			if err != nil {
//FIXME
				return
			}
			w.Write([]byte("["))
			for idx, id := range list {
				if idx < len(list) - 1 {
					fmt.Fprintf(w, "%v,", id)
				} else {
					fmt.Fprintf(w, "%v", id)
				}
			}
			w.Write([]byte("]"))
		} else if req.Method == "POST" {
			paths := strings.Split(req.RequestURI, "/")
			if len(paths) < 5 {
//FIXME
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if paths[4] == "submit" {
				id, err := strconv.Atoi(paths[3])
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				log.Printf("Got submit request for %v %v", user, id)
				bm.BundleStateSet(user, id, BundleState_ToBundle)
			} else if paths[4] == "delete" {
				id, err := strconv.Atoi(paths[3])
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				log.Printf("Got delete request for %v %v", user, id)
//FIXME allow the service user to specify the watcher.
				bm.BundleDelete(user, "web", id)
			}
		} else if req.Method == "GET" {
			paths := strings.Split(req.RequestURI, "/")
			id, err := strconv.Atoi(paths[3])
			if err != nil {
				panic(err)
			}
			b, err := bm.BundleGet(user, id)
			panic_err(err)
			state, err := b.StateGet()
			panic_err(err)
			available, err := b.AvailableGet()
			panic_err(err)
			transaction, err := b.TransactionGet()
			panic_err(err)
			errormsg, err := b.ErrorGet()
			panic_err(err)
			file_service, err := b.FileServiceGet()
			panic_err(err)
			flist, err := b.FilesGet()
			panic_err(err)
			fstr := ""
			for id, file := range flist {
				if id > 0 {
					fstr += ","
				}
				spacifica_filename, err := file.PacificaFilenameGet()
				panic_err(err)
				pacifica_filename, err := json.Marshal(spacifica_filename)
				panic_err(err)
				slocal_filename, err := file.LocalFilenameGet()
				panic_err(err)
				local_filename, err := json.Marshal(slocal_filename)
				panic_err(err)
				sdisable_on_error_msg, err := file.DisableOnErrorMsgGet()
				panic_err(err)
				disable_on_error, err := file.DisableOnErrorGet()
				panic_err(err)
				disable_on_error_msg, err := json.Marshal(sdisable_on_error_msg)
				panic_err(err)
				mtime, err := file.MtimeGet()
				panic_err(err)
				smtime := "null"
				if mtime != nil {
					bmtime, err := mtime.MarshalJSON()
					panic_err(err)
					smtime = string(bmtime)
				}
				groups, err := file.GroupsGet()
				panic_err(err)
				sgroups := ""
				for id, group := range groups {
					if id > 0 {
						sgroups += ","
					}
					t, err := json.Marshal(group[0])
					panic_err(err)
					n, err := json.Marshal(group[1])
					panic_err(err)
					sgroups += fmt.Sprintf("{\"Type\":%v, \"Name\":%v}", string(t), string(n))
				}
//FIXME verify json encodings.
				fstr += fmt.Sprintf(`
{
	"Id":%v,
	"PacificaFilename":%v,
	"LocalFilename":%v,
	"Mtime":%v,
	"DisableOnErrorMsg":%v,
	"DisableOnError":%v,
	"Groups":[%v]
}`,
	file.id,
	string(pacifica_filename),
	string(local_filename),
	smtime,
	string(disable_on_error_msg),
	disable_on_error,
	sgroups)
			}
			fmt.Fprintf(w, `{
	"State":%v,
	"Available":%v,
	"FileService":"%v",
	"Transaction":"%v",
	"ErrorMsg":"%v",
	"Submittable":%v,
	"Editable":%v,
	"MakeEditable":%v,
	"Deletable":%v,
	"Cancelable":%v,
	"Files":[%v]

}`,
	state,
	available,
	file_service,
	transaction,
	errormsg,
	BundleStateActionOk(state, BundleAction_Submittable),
	BundleStateActionOk(state, BundleAction_Editable),
	BundleStateActionOk(state, BundleAction_MakeEditable),
	BundleStateActionOk(state, BundleAction_Deletable),
	BundleStateActionOk(state, BundleAction_Cancelable),
	fstr)
		}
	}
	return
}

func bundleManagerInit() *BundleManager {
	bm = BundleManagerNew()
	if bm == nil {
		return nil
	}
	web.ServMux.HandleFunc("/bundle/json/", func(w http.ResponseWriter, req *http.Request) { bundleIdsHandle(bm, w, req) })
	return bm
}
