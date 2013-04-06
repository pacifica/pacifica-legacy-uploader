package upload

import (
	"io"
	"os"
	"log"
	"path"
	"sync"
	"time"
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"platform"
	"path/filepath"
	"encoding/json"
	"pacificauploaderd/common"
	"pacificauploaderuserd/rpc"
)

type bundler struct {
	wake *sync.Cond
	bm *BundleManager
}

type metadataGroups struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type metadataFileEntry struct {
	Sha1 string `json:"sha1Hash"`
	Subdir string `json:"destinationDirectory"`
	Filename string `json:"fileName"`
	LocalFilePath string `json:"localFilePath"`
	Mtime time.Time `json:"mtime"`
	Groups []metadataGroups `json:"groups"`
}

type metadataFile struct {
	Version string `json:"version"`
	Uuid string `json:"clientuuid"`
	Files []metadataFileEntry `json:"file"`
}

func writeMetadata(bm *BundleManager, user string, bundle_id int, writer io.Writer) error {
	b, err := bm.BundleGet(user, bundle_id)
	if err != nil {
		return err
	}
	metadata := &metadataFile{Version:"1.0.0", Uuid:common.Uuid}
	bfs, err := b.FilesGet()
	if err != nil {
		log.Printf("Failed to get files from bundle.\n")
		return nil
	}
	for _, bf := range bfs {
		disabled_on_error_mesg, err := bf.DisableOnErrorMsgGet()
		if err != nil {
			return err
		}
		if disabled_on_error_mesg != "" {
			continue
		}
		pacifica_filename, err := bf.PacificaFilenameGet()
		if err != nil {
			return err
		}
		local_filename, err := bf.LocalFilenameGet()
		if err != nil {
			return err
		}
		filename := path.Base(pacifica_filename)
		subdir := path.Dir(pacifica_filename)
		if subdir == "." {
			subdir = ""
		}
		if strings.HasPrefix(subdir, "/") {
			subdir = subdir[1:]
		}
		sha1, err := bf.Sha1Get()
		if err != nil {
			return err
		}
		mtime, err := bf.MtimeGet()
		if err != nil {
			return err
		}
		groups, err := bf.GroupsGet()
		if err != nil {
			return err
		}
		mdfe := metadataFileEntry{LocalFilePath: local_filename, Sha1: sha1, Subdir: subdir, Filename: filename, Mtime: *mtime}
		for _, group := range groups {
			mdfe.Groups = append(mdfe.Groups, metadataGroups{Type:group[0], Name:group[1]})
		}
		metadata.Files = append(metadata.Files, mdfe)
	}
	log.Printf("Bundle files remain %v\n", len(metadata.Files))
	if len(metadata.Files) <= 0 {
		err = bm.BundleStateSet(user, bundle_id, BundleState_Error)
		if err != nil {
			log.Printf("Failed to transition bundle to success state. %v\n", err)
			return err
		}
		return errors.New("Bundle Done")
	}
	j, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	buff := new(bytes.Buffer)
	json.Indent(buff, j, "", "\t")
	_, err = writer.Write(buff.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func bundleFiles(bundle_filename string, user string, bundle_id int) error {
	b, err := bm.BundleGet(user, bundle_id)
	if err != nil {
		return err
	}
	bfs, err := b.FilesGet()
	if err != nil {
		log.Printf("Failed to get files from bundle.\n")
		return nil
	}
	for _, bf := range bfs {
		pacifica_filename, err := bf.PacificaFilenameGet()
		if err != nil {
			return err
		}
		if strings.HasPrefix(pacifica_filename, "/") {
			pacifica_filename = pacifica_filename[1:]
		}
		local_filename, err := bf.LocalFilenameGet()
		if err != nil {
			return err
		}
		disable_on_error, err := bf.DisableOnErrorGet()
		if err != nil {
			return err
		}
		log.Printf("To bundle: %v %v %v\n", bundle_filename, local_filename, pacifica_filename)
		result, sha1, fe, mtime, err := common.UserBundleFile(user, bundle_filename, local_filename, pacifica_filename)
		if err != nil {
			log.Printf("Failed to talk to the userd bundler. %v\n", err)
			return err
		}
		if rpc.BundleFileErrorIsError(fe) {
			if rpc.BundleFileErrorIsTransient(fe, !disable_on_error) {
				log.Printf("Transient issue. %v\n", fe)
				return errors.New("Transient error.")
			} else if disable_on_error && rpc.BundleFileErrorIsTransientError(fe) {
//FIXME better error message.
				log.Printf("Transient error. %v\n", fe)
				err = bm.BundleFileStringSet(user, bf.bundle.id, bf.id, "disable_on_error_msg", strconv.Itoa(int(fe)))
				if err != nil {
					return err
				}
				continue
			} else if rpc.BundleFileErrorIsPermanentError(fe, !disable_on_error) {
				log.Printf("Permanent error. %v\n", fe)
				err = bm.BundleStateSet(user, bundle_id, BundleState_Error)
				if err != nil {
					log.Printf("Failed to transition bundle to error state. %v\n", err)
					return err
				}
				err = bm.bundleStringSet(user, bundle_id, "error", "Bundle Error")
				if err != nil {
					log.Printf("Failed to set error message. %v\n", err)
					return err
				}
//FIXME better error message.
				err = bm.BundleFileStringSet(user, bf.bundle.id, bf.id, "disable_on_error_msg", strconv.Itoa(int(fe)))
				if err != nil {
					return err
				}
				return errors.New("PermanentError")
			}
		}
		err = bf.Sha1Set(sha1)
		if err != nil {
			log.Printf("Failed to set sha1. %v\n", err)
			return errors.New("Sha1")
		}
		err = bf.MtimeSet(mtime)
		if err != nil {
			log.Printf("Failed to set  mtime. %v\n", err)
			return errors.New("Mtime")
		}
		log.Printf("Got from bundler: %v %v %v\n", fe, result, sha1)
	}
	return nil
}

func bundlerNew(bm *BundleManager) *bundler {
	self := &bundler{bm: bm, wake: sync.NewCond(&sync.Mutex{})}
	err := bm.BundleStateChangeWatchSet(BundleState_ToBundle, func(user string, bundle_id int, state BundleState) { self.Wakeup() })
	if err != nil {
		log.Printf("Failed to register wakeup callback with the bundle manager.")
		return nil
	}
	go func() {
		for {
			found := false
			ids, err := self.bm.bundleIdsForStateGet(BundleState_ToBundle)
			if err == nil && ids != nil {
				for user, list := range ids {
					for _, id := range list {
						log.Printf("Got bundles %v %v\n", user, id)
						bundle_filename := filepath.Join(common.BaseDir, "bundles", "inprogress", strconv.Itoa(id) + ".tar")
						file, err := os.Create(bundle_filename)
						if err != nil {
							log.Printf("Could not open temp file. %v", err)
							continue
						}
						file.Close()
						if common.System {
							if platform.PlatformGet() == platform.Linux {
								cmd := exec.Command("setfacl", "-m", "user:" + user + ":rw", bundle_filename)
								err = cmd.Run()
								if err != nil {
									log.Printf("Failed to set acl on bundle. %v\n", err)
									continue
								}
							} else if platform.PlatformGet() == platform.Windows {
								//TODO - remove or fix so that permissions are set for system user only.
								/*userstr := common.UserdDefaultUsername() + ":C"
								err = common.Cacls(bundle_filename, "/p", "NT AUTHORITY\\SYSTEM:f", userstr, "BUILTIN\\Administrators:F")
								if err != nil {
									log.Printf("Failed to run cacls %v\n", err)
									continue
								}*/
							}
						}
						err = bundleFiles(bundle_filename, user, id)
						if err != nil {
							continue
						}
						var newstate BundleState
						metadata_filename := filepath.Join(common.BaseDir, "bundles", "inprogress", strconv.Itoa(id) + ".txt")
						file, err = os.Create(metadata_filename)
						if err != nil {
							log.Printf("Could not open temp file. %v", err)
							continue
						} else {
							err = writeMetadata(self.bm, user, id, file)
							file.Close()
							if err != nil {
								continue
							}
							_, _, fe, _, err := common.UserBundleFile(user, bundle_filename, metadata_filename, "metadata.txt")
							if err != nil || rpc.BundleFileErrorIsError(fe) {
								log.Printf("Failed to add metadata to bundle. %v", err)
								continue
							}
							newstate = BundleState_ToUpload
						}
						if common.System {
							if platform.PlatformGet() == platform.Linux {
								cmd := exec.Command("setfacl", "-x", "user:" + user, bundle_filename)
								err = cmd.Run()
								if err != nil {
									log.Printf("Failed to remove acl on bundle. %v\n", err)
									continue
								}
							} else if platform.PlatformGet() == platform.Windows {
								err = common.Cacls(bundle_filename, "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Administrators:F")
								if err != nil {
									log.Printf("Failed to run cacls %v\n", err)
									continue
								}
							}
						}
						bundle_new_filename := filepath.Join(common.BaseDir, "bundles", strconv.Itoa(id) + ".tar")
						os.Remove(bundle_new_filename)
						err = os.Rename(bundle_filename, bundle_new_filename)
						if err != nil {
							log.Printf("Failed to rename bundle. %v\n", err)
							continue
						}
						err = bm.bundleStringSet(user, id, "bundle_location", bundle_new_filename)
						if err != nil {
							log.Printf("Failed to set bundle location. %v\n", err)
							continue
						}
						err = bm.BundleStateSet(user, id, newstate)
						if err != nil {
							log.Printf("Failed to transition bundle from bundling to upload. %v\n", err)
							continue
						}
						found = true
					}
				}
			}
			if found == false {
				self.wake.L.Lock()
				count, err := self.bm.bundleIdsForStateCount(BundleState_ToBundle)
				if err == nil && count < 1 {
					self.wake.Wait()
				}
				self.wake.L.Unlock()
			}
		}
	} ()
	return self
}

//Must not be called with sql lock held
func (self *bundler) Wakeup() {
	self.wake.L.Lock()
	self.wake.Signal()
	self.wake.L.Unlock()
}

func bundlerInit() {
	os.MkdirAll(filepath.Join(common.BaseDir, "bundles"), 0755)
	err := os.RemoveAll(filepath.Join(common.BaseDir, "bundles", "inprogress"))
	if err != nil {
		log.Printf("Failed to clean out the inprogress. %v\n", err)
		os.Exit(-1)
	}
	os.MkdirAll(filepath.Join(common.BaseDir, "bundles", "inprogress"), 0755)
	if common.System && platform.PlatformGet() == platform.Windows {
		err = common.Cacls(filepath.Join(common.BaseDir, "bundles", "inprogress"), "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Users:r", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls %v\n", err)
			os.Exit(-1)
		}
	}
}
