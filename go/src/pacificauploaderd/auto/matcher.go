package auto

import (
	"log"
	"pacificauploaderd/common"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type matcher struct {
	reversePaths *map[string][]ReverseLookupEntry
}

type filePathSlice []string

func (fp filePathSlice) Len() int {
	return len(fp)
}

func (fp filePathSlice) Less(i int, j int) bool {
	res := len(fp[i]) - len(fp[j])
	if res == 0 {
		return fp[i] < fp[j]
	}
	return res < 0
}

func (fp filePathSlice) Swap(i int, j int) {
	fp[i], fp[j] = fp[j], fp[i]
}

func (fp filePathSlice) Sort() {
	sort.Sort(fp)
}

type patternMapEntry struct {
	id    int
	value string
}

//TODO - break this function up into more digestable bits, it's deeply nested and painful to read.
func (self *matcher) found(path string, fullpath string) {
	//TODO - remove...
	//common.Dprintf("Found path: %s fullpath: %s\n", path, fullpath)
	fileinfo, err := os.Lstat(fullpath)
	if err != nil {
		log.Printf("Failed to stat %s\n", fullpath)
		return
	}
	reversePaths := *self.reversePaths
	rlea := reversePaths[path]
	if rlea == nil {
		return
	}
	if len(rlea) < 1 {
		common.Dprintf("No reverse lookup entries found for %s", path)
	}

	//Process each ReverseLookupEntry set for path
	for _, rle := range rlea {
		if rle.Rule != nil && (rle.Prefix == "" || strings.HasPrefix(fullpath, rle.Prefix+string(os.PathSeparator)) || fullpath == rle.Prefix) {
			//Determine if the fullpath matches the exclude pattern for the corresponding WatchRule.			
			excluded := false
			for i := 0; i < len(rle.Rule.ExcludePatterns); i++ {
				//FIXME This can perform better by precompiling patterns and putting them in rle. Consider doing this later.
				matched, err := regexp.Match(rle.Rule.ExcludePatterns[i], []byte(fullpath))
				if err != nil || matched == false {
					continue
				}
				excluded = true
			}
			if excluded {
				common.Dprintf("Excluding %s %s %s", rle.User, rle.Rule.Name, fullpath)
				continue
			}

			//Get the fileState for this path
			filestate, err := fsm.getFileState(rle.User, rle.Rule.Name, fullpath)
			if err != nil {
				log.Printf("Failed to get a fileState for %s, %s, %s, error %v",
					rle.User, rle.Rule.Name, fullpath, err)
				continue
			}

			//The file has changed, it should be processed (a.k.a matched)
			if checkFileChanged(fullpath, fileinfo, filestate) {
				ok, err := common.UserAccess(rle.User, fullpath)
				if err != nil {
					log.Printf("User access to %v could not be granted", err)
				}
				if ok {
					groups := make([][2]string, 0)
					for _, sg := range rle.Rule.StaticMetadata {
						groups = append(groups, [2]string{sg.Type, sg.Name})
					}

					//TODO - NGT 4/25/12 This is difficult to read.  It looks like it is both determining if the file matches
					//and generating groups.  A great candidate for refactoring into one or two functions...
					//Aha, after reading more closely it appears the groups list is being "appended" (or potentially replaced) with additional
					//type/name pairs based on groups that were passed through regular expressions in the WatchRules.
					//If so, this needs to be clearly explained in comments.  Also, we need to document in the UI somewhere the
					//regular expressions format allowed by the regex engine (even if it's just a link) or a couple canned examples...
					//This code is nothing short of magic :)
					for _, rule := range rle.Rule.MetadataPattern {
						common.Dprintf("Got pattern %v\n", rule.Pattern)
						//FIXME This can perform better by precompiling patterns and putting them in rle. Consider doing this later.
						re, err := regexp.Compile(rule.Pattern)
						if err != nil {
							break
						}
						groupMap := make(map[string]string)
						for _, x := range rule.Group {
							groupMap[x.Pattern] = x.Value
						}
						subexpnames := re.SubexpNames()
						patternMapList := []patternMapEntry{}
						for id, x := range subexpnames {
							if x != "" && groupMap[x] != "" {
								common.Dprintf("Subexp: %v %v\n", x, groupMap[x])
								patternMapList = append(patternMapList, patternMapEntry{id, groupMap[x]})
							}
						}
						match := re.FindStringSubmatch(fullpath)
						if match != nil {
							common.Dprintf("%s matched.\n", fullpath)
							for _, x := range patternMapList {
								common.Dprintf("Map: %v %v %v\n", x.value, match[x.id], fullpath)
								groups = append(groups, [2]string{x.value, match[x.id]})
							}
						}
					}

					//TODO - another block that is a good candidate for refactor into a function
					//Generate the new file name based on the rename patterns
					var newfilename string = fullpath
					for _, rule := range rle.Rule.RenamePatterns {
						re, err := regexp.Compile(rule.Pattern)
						if err != nil {
							break
						}
						match := re.FindStringSubmatchIndex(fullpath)
						if match != nil {
							n := re.ExpandString([]byte{}, rule.Value, newfilename, match)
							if n != nil {
								common.Dprintf("Renaming %s %s\n", newfilename, string(n))
								newfilename = string(n)
							}
						}
					}
					var tmpprefix string
					if rle.Prefix == "" {
						tmpprefix = path
					} else {
						tmpprefix = rle.Prefix
					}
					tmpprefix += string(os.PathSeparator)
					if strings.HasPrefix(newfilename, tmpprefix) {
						newfilename = newfilename[len(tmpprefix):]
					}
					newfilename = strings.Replace(newfilename, string(os.PathSeparator), "/", -1)

					am.addAutoFile(*filestate, newfilename, groups)
				}
			} else {
				common.Dprintf("checkFileChanged(%s, %+v, %+v) returned false skipping...", fullpath, fileinfo, filestate)
			}
		} else {
			common.Dprintf("reverse lookup entry %+v was skipped for processing", rle)
		}
	}
}

//TODO - break this function up into more digestable bits, it's deeply nested and painful to read.
func (self *matcher) updatePaths(allConfigs *map[string]*UserConfig) {
	reversePaths := map[string][]ReverseLookupEntry{}
	paths := map[string]bool{}
	for user, uc := range *allConfigs {
		for _, rule := range uc.Rules {
			for _, path := range rule.Paths {
				paths[path] = true
				if reversePaths[path] == nil {
					reversePaths[path] = make([]ReverseLookupEntry, 0)
				}
				reversePaths[path] = append(reversePaths[path], ReverseLookupEntry{User: user, Rule: rule})
			}
		}
	}
	var newpaths filePathSlice
	for key, _ := range paths {
		newpaths = append(newpaths, key)
	}
	newpaths.Sort()
	var valid []bool = make([]bool, len(newpaths))
	for i := 0; i < len(newpaths); i++ {
		valid[i] = true
	}
	for i := 0; i < len(newpaths); i++ {
		path := newpaths[i]
		common.Dprintf("newpath - %d %s\n", i, path)
		for j := i + 1; j < len(newpaths); j++ {
			if valid[j] && strings.HasPrefix(newpaths[j], path+string(os.PathSeparator)) {
				common.Dprintf("Collapsable - %s\n", newpaths[j])
				rlea := reversePaths[newpaths[j]]
				if rlea != nil {
					for _, rle := range rlea {
						rle.Prefix = newpaths[j]
						reversePaths[path] = append(reversePaths[path], rle)
					}
				}
				valid[j] = false
			}
			if valid[j] {
				common.Dprintf("cmpnewpath - %s %s %t\n", path, newpaths[j], strings.HasPrefix(newpaths[j], path+string(os.PathSeparator)))
			}
		}
	}
	var tmpnewpaths filePathSlice
	tmpReversePaths := map[string][]ReverseLookupEntry{}
	for i := 0; i < len(newpaths); i++ {
		if valid[i] {
			tmpnewpaths = append(tmpnewpaths, newpaths[i])
			tmpReversePaths[newpaths[i]] = reversePaths[newpaths[i]]
		}
	}
	newpaths = tmpnewpaths
	reversePaths = tmpReversePaths
	for path, rlea := range reversePaths {
		if rlea != nil {
			for _, rle := range rlea {
				if rle.Rule != nil {
					common.Dprintf("rle - %s %s %s %s\n", path, rle.User, rle.Rule.Name, rle.Prefix)
				}
			}
		}
	}
	if w != nil {
		w.updatePaths([]string(newpaths))
	}
	common.Dprintf("Unique paths: %v\n", newpaths)
	self.reversePaths = &reversePaths
}

func checkFileChanged(path string, file os.FileInfo, filestate *fileState) bool {
	var res bool
	if file.IsDir() {
		log.Printf("%s is a directory.", path)
		return false
	}
	if path == "" {
		log.Printf("checkedFileChanged was called with an empty path")
		return false
	}
	if filestate.passOff != notSeenBefore {
		if file.ModTime().UnixNano() != filestate.lastModified {
			filestate.passOff = notSeenBefore
		}
	}
	if filestate.passOff == notSeenBefore {
		res = true
	}
	filestate.lastSeen = time.Now().UnixNano()
	filestate.lastModified = file.ModTime().UnixNano()
	return res
}

func matcherInit() {
	m = new(matcher)
	w.found = append(w.found, func(a string, b string) { m.found(a, b) })
}
