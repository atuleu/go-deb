package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"launchpad.net/go-xdg"

	deb ".."

	"github.com/nightlyone/lockfile"
)

type XdgArchiver struct {
	basepath string
	lock     lockfile.Lockfile
	auth     DebfileAuthentifier
}

var xaLockName = "go-deb.ddesk/archives/global.lock"

func NewXdgArchiver(auth DebfileAuthentifier) (*XdgArchiver, error) {
	lockpath, err := xdg.Data.Ensure(xaLockName)
	if err != nil {
		return nil, err
	}

	res := &XdgArchiver{
		basepath: path.Dir(lockpath),
		auth:     auth,
	}
	res.lock, err = lockfile.New(lockpath)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (a *XdgArchiver) tryLock() error {
	if err := a.lock.TryLock(); err != nil {
		return fmt.Errorf("Could not lock %s: %s", a.basepath, err)
	}
	return nil
}

func (a *XdgArchiver) unlockOrPanic() {
	if err := a.lock.Unlock(); err != nil {
		panic(err)
	}
}

func (a *XdgArchiver) ensureSourceChanges(p deb.SourceControlFile, files []string) (*deb.ChangesFile, []string, error) {
	//we create a temp directory that will hold the extracted data:
	tmpDir, err := ioutil.TempDir("", "xdg-archiver-source-package_")
	if err != nil {
		return nil, files, fmt.Errorf("Could not create a working directory: %s", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			panic(err)
		}
	}()

	//test if a source changes exists
	sChangesPath := path.Join(p.BasePath, p.ChangesFilename())

	exists, err := a.fileExists(sChangesPath)
	if err != nil {
		return nil, files, err
	}

	if exists {
		f, err := os.Open(sChangesPath)
		if err != nil {
			return nil, files, err
		}
		unsigned, err := a.auth.CheckAnyClearsigned(f)
		if err != nil {
			return nil, files, err
		}

		c, err := deb.ParseChangeFile(unsigned)
		if err != nil {
			return nil, files, err
		}
		return c, files, nil
	}

	err = a.copyListOfFile(p.BasePath, tmpDir, files)
	if err != nil {
		return nil, files, err
	}

	cmd := exec.Command("dpkg-source", "-x",
		p.Filename())
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, files, fmt.Errorf("Could not extract source package:\n%s", output)
	}

	extractedDir := path.Join(tmpDir, fmt.Sprintf("%s-%s", p.Identifier.Source, p.Identifier.Ver.UpstreamVersion))
	cmd = exec.Command("dpkg-genchanges", "-S")
	var toParse, changes, out bytes.Buffer
	cmd.Stdout = io.MultiWriter(&changes, &toParse)
	cmd.Stderr = &out
	cmd.Dir = extractedDir
	err = cmd.Run()
	if err != nil {
		return nil, files, fmt.Errorf("Could not generate .changes file :\n%s", out.String())
	}

	res, err := deb.ParseChangeFile(&toParse)
	if err != nil {
		return nil, files, err
	}

	err = ioutil.WriteFile(sChangesPath, changes.Bytes(), 0644)
	if err != nil {
		return nil, files, fmt.Errorf("Could not write %s: %s", sChangesPath, err)
	}

	files = append(files, path.Base(sChangesPath))

	err = a.auth.SignChanges(sChangesPath)
	if err != nil {
		return nil, files, err
	}

	return res, files, nil

}

func (a *XdgArchiver) makeAbrevation(source string) string {
	if len(source) == 0 {
		panic("Should not be called with an empty string")
	}
	lowered := strings.ToLower(source)
	if strings.HasPrefix(lowered, "lib") && len(source) > 3 {
		return "lib" + string(lowered[3])
	}
	return string(lowered[0])
}

func (a *XdgArchiver) sourceStorePath(p deb.SourcePackageRef) (string, error) {
	if len(p.Source) == 0 {
		return "", fmt.Errorf("Package name should not be empty")
	}

	//we check for all files
	key := strings.ToLower(p.Source)
	return path.Join(a.basepath, "sources", a.makeAbrevation(key), key), nil
}

func (a *XdgArchiver) binaryStorePath(p deb.SourcePackageRef) (string, error) {
	if len(p.Source) == 0 {
		return "", fmt.Errorf("package source should not be empty")
	}
	key := strings.ToLower(p.Source)
	return path.Join(a.basepath, "binary", a.makeAbrevation(key), key), nil
}

func (a *XdgArchiver) sourceJsonName(p deb.SourcePackageRef) string {
	return p.String() + ".source.json"
}

func (a *XdgArchiver) binaryJsonName(p deb.SourcePackageRef) string {
	return p.String() + ".binary.json"
}

func (a *XdgArchiver) copyFile(inPath, outPath string) error {
	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("Could not copy %s to %s:  %s", inPath, outPath, err)
	}

	err = out.Sync()
	if err != nil {
		return fmt.Errorf("Could not sync %s: %s", outPath, err)
	}

	err = out.Close()
	if err != nil {
		return fmt.Errorf("Could not close %s: %s", outPath, err)
	}

	return nil
}

func (a *XdgArchiver) fileExists(f string) (bool, error) {
	_, err := os.Stat(f)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("Could not check existence of file %s: %s", f, err)
	}
	return true, nil
}

func (a *XdgArchiver) compareByteSlice(aa, bb []byte) bool {
	if len(aa) != len(bb) {
		return false
	}
	for i, v := range aa {
		if bb[i] != v {
			return false
		}
	}
	return true
}

func (a *XdgArchiver) fileMd5(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	h := md5.New()
	if _, err = io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("Could not compute md5 sum of %s: %s", path, err)
	}
	return h.Sum(nil), nil
}

func (a *XdgArchiver) listAndCheckFiles(p deb.SourceControlFile) ([]string, error) {
	toCopy := []string{}

	dest, err := a.sourceStorePath(p.Identifier)
	if err != nil {
		return toCopy, err
	}

	for _, f := range p.Md5Files {
		err := f.CheckFile(p.BasePath, md5.New())
		if err != nil {
			return toCopy, fmt.Errorf("Could not check %s: %s", f.Name, err)
		}
		if strings.Contains(f.Name, ".orig.tar") == false {
			toCopy = append(toCopy, f.Name)
			continue
		}
		//we will not copy the file, only check the md5 sum of the current one
		finalOut := path.Join(dest, f.Name)
		exists, err := a.fileExists(finalOut)
		if err != nil {
			return toCopy, err
		}

		if exists == false {
			toCopy = append(toCopy, f.Name)
			continue
		}

		cs, err := a.fileMd5(finalOut)
		if err != nil {
			return toCopy, err
		}
		if a.compareByteSlice(cs, f.Checksum) == false {
			return toCopy, fmt.Errorf("File %s already exists and has different checksum", finalOut)
		}
	}
	toCopy = append(toCopy, p.Filename())

	exists, err := a.fileExists(path.Join(p.BasePath, p.ChangesFilename()))
	if err != nil {
		return toCopy, err
	}
	if exists == true {
		toCopy = append(toCopy, p.ChangesFilename())
	}

	return toCopy, nil
}

func (a *XdgArchiver) copyListOfFile(inPath, outPath string, files []string) error {
	if inPath == outPath {
		return nil
	}
	for _, f := range files {
		inFile := path.Join(inPath, f)
		outFile := path.Join(outPath, f)

		if err := a.copyFile(inFile, outFile); err != nil {
			return err
		}
	}
	return nil
}

func (a *XdgArchiver) copySourceFilesToStage(p deb.SourceControlFile) ([]string, string, error) {
	dest, err := a.sourceStorePath(p.Identifier)
	if err != nil {
		return []string{}, dest, err
	}
	// first stage the file
	dest = path.Join(dest, "stage")

	toCopy, err := a.listAndCheckFiles(p)
	if err != nil {
		return toCopy, dest, err
	}

	//makes sure directory exists !
	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return toCopy, dest, err
	}

	err = a.copyListOfFile(p.BasePath, dest, toCopy)

	return toCopy, dest, err
}

func (a *XdgArchiver) ArchiveSource(p deb.SourceControlFile) (*ArchivedSource, error) {
	if err := a.tryLock(); err != nil {
		return nil, err
	}
	defer a.unlockOrPanic()

	files, dest, err := a.copySourceFilesToStage(p)
	if err != nil {
		return nil, err
	}
	// whatever happen, we destroy the stage dir
	defer os.RemoveAll(dest)

	p.BasePath = dest
	changes, files, err := a.ensureSourceChanges(p, files)
	if err != nil {
		return nil, err
	}

	//now that everything is fine and signed, we move it to the final destination
	finalDest := path.Dir(dest)
	for _, f := range files {
		fTarget := path.Join(dest, f)
		fDest := path.Join(finalDest, f)
		err := os.Rename(fTarget, fDest)
		if err != nil {
			return nil, err
		}
	}
	p.BasePath = finalDest

	res := &ArchivedSource{
		Changes:  changes,
		Dsc:      p,
		BasePath: finalDest,
	}

	//marshallize the data in a json object put in the directory for fast recovery
	jsonDataPath := path.Join(finalDest, a.sourceJsonName(p.Identifier))
	f, err := os.Create(jsonDataPath)
	if err != nil {
		return nil, err
	}
	enc := json.NewEncoder(f)
	err = enc.Encode(res)
	if err != nil {
		return nil, fmt.Errorf("Could not save archive data: %s", err)
	}
	return res, nil
}

func (a *XdgArchiver) ArchiveBuildResult(b BuildResult) (*BuildResult, error) {
	if err := a.tryLock(); err != nil {
		return nil, err
	}
	defer a.unlockOrPanic()

	//we ensure that the _binaries.changes exists
	changesPath := path.Join(b.BasePath, b.ChangesPath)
	exists, err := a.fileExists(changesPath)
	if err != nil {
		return nil, err
	}
	if exists == false {
		return nil, fmt.Errorf("Missing required file %s", changesPath)
	}

	destPath, err := a.binaryStorePath(b.Changes.Ref.Identifier)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(destPath, 0755)
	if err != nil {
		return nil, err
	}

	for _, f := range b.Changes.Md5Files {
		err := f.CheckFile(b.BasePath, md5.New())
		if err != nil {
			return nil, err
		}
		sourcePath := path.Join(b.BasePath, f.Name)
		destPath := path.Join(destPath, f.Name)
		err = a.copyFile(sourcePath, destPath)
		if err != nil {
			return nil, err
		}
	}
	finalChangesPath := path.Join(destPath, b.ChangesPath)
	err = a.copyFile(changesPath, finalChangesPath)
	if err != nil {
		return nil, err
	}

	b.BasePath = destPath

	err = a.auth.SignChanges(finalChangesPath)
	if err != nil {
		return nil, err
	}

	jsonDataPath := path.Join(destPath, a.binaryJsonName(b.Changes.Ref.Identifier))
	f, err := os.Create(jsonDataPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(b)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (a *XdgArchiver) GetArchivedSource(p deb.SourcePackageRef) (*ArchivedSource, error) {
	if err := a.tryLock(); err != nil {
		return nil, err
	}
	defer a.unlockOrPanic()

	dest, err := a.sourceStorePath(p)
	if err != nil {
		return nil, err
	}

	jsonFile := path.Join(dest, a.sourceJsonName(p))
	exists, err := a.fileExists(jsonFile)
	if err != nil {
		return nil, err
	}

	if exists == false {
		return nil, fmt.Errorf("%s is not archived", p)
	}

	f, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}

	d := json.NewDecoder(f)
	res := &ArchivedSource{}
	return res, d.Decode(res)
}

func (a *XdgArchiver) GetBuildResult(p deb.SourcePackageRef) (*BuildResult, error) {
	if err := a.tryLock(); err != nil {
		return nil, err
	}
	defer a.unlockOrPanic()

	basePath, err := a.binaryStorePath(p)
	if err != nil {
		return nil, err
	}

	jsonDataPath := path.Join(basePath, a.binaryJsonName(p))
	f, err := os.Open(jsonDataPath)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(f)
	res := &BuildResult{}
	err = dec.Decode(res)
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve json static data: %s", err)
	}

	return res, nil
}

func init() {
	aptDepTracker.Add("dpkg-dev")
	aptDepTracker.Add("devscripts")
}
