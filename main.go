package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

var toHash = flag.String("toHash", "", "Comma-separated list of files to consider when building the hash")
var pkgDirs = flag.String("packageDirectories", "", "Comma-separated list of directories to archive to build the cache")
var cmd = flag.String("buildCommand", "", "Command to run which will build the package directories")
var bucketName = flag.String("bucketName", "", "Bucket to store artifacts in s3")
var outfile = flag.String("outfile", "out.tar.gz", "Name of the tarball that should be generated")

func hashFiles(fileCsv string) hash.Hash {
	files := strings.Split(fileCsv, ",")
	hash := md5.New()
	for _, f := range files {
		file, err := os.Open(f)
		handleErr(err)
		_, err = io.Copy(hash, file)
		handleErr(err)
	}
	log.Printf("Ended up with: %x", hash.Sum(nil))
	return hash
}

func buildPath(hash string) string {
	return strings.Join([]string{"build", hash}, "-")
}

func build(cmd string) {
	log.Printf("Running: /bin/bash -C \"%s\"", cmd)
	c := exec.Command("/bin/bash", "-c", cmd)
	err := c.Run()
	handleErr(err)
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func archive(pkgDirs, writeTo string) {
	out, err := os.Create(writeTo)
	handleErr(err)
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	dirs := strings.Split(pkgDirs, ",")
	for _, dir := range dirs {
		walk(dir, tw)
	}

	log.Printf("%s written.\n", out.Name())
}

func walk(dirPath string, tw *tar.Writer) {
	dir, err := os.Open(dirPath)
	handleErr(err)
	defer dir.Close()
	fis, err := dir.Readdir(0)
	handleErr(err)
	for _, fi := range fis {
		curPath := dirPath + "/" + fi.Name()
		if fi.IsDir() {
			walk(curPath, tw)
		} else {
			iterWriteTar(curPath, tw)
		}
	}
}

func iterWriteTar(path string, tw *tar.Writer) {
	file, err := os.Open(path)
	handleErr(err)
	defer file.Close()

	fi, err := os.Lstat(path)
	handleErr(err)

	isSym := (fi.Mode() & os.ModeSymlink) != 0
	symPath := ""

	if isSym {
		symPath, err = os.Readlink(path)
		handleErr(err)
	}

	h, err := tar.FileInfoHeader(fi, symPath)
	handleErr(err)

	h.Name = file.Name() // maybe?
	if !isSym {
		h.Size = fi.Size()
	}

	b, err := ioutil.ReadAll(file)
	handleErr(err)

	err = tw.WriteHeader(h)
	handleErr(err)

	if !isSym {
		_, err = tw.Write(b)
		handleErr(err)
	}

	err = tw.Flush()
	handleErr(err)
}

func upload(bucket *s3.Bucket, checksum, filename string) {
	f, err := os.Open(filename)
	handleErr(err)
	defer f.Close()
	path := fmt.Sprintf("%s/%s", checksum, filename)
	fi, err := f.Stat()
	handleErr(err)
	err = bucket.PutReader(path, f, fi.Size(), "binary/octet-stream", s3.PublicRead)
	if err != nil {
		log.Fatalf("Go makes me sad because of: %s", err)
	}
	log.Printf("Uploaded %s", path)
}

func extract(file []byte) {
	gr, err := gzip.NewReader(bytes.NewReader(file))
	handleErr(err)

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // end of archive
		}
		handleErr(err)
		// TODO(justinabrahms): Handle the case for directories.
		dirName := path.Dir(hdr.Name)
		os.MkdirAll(dirName, 0777)
		f, err := os.Create(hdr.Name)
		handleErr(err)
		defer f.Close()
		io.Copy(f, tr)
	}
}

func main() {
	flag.Parse()

	if *toHash == "" || *pkgDirs == "" || *cmd == "" || *bucketName == "" {
		log.Print("You are missing one or more mandatory command line arguments.")
		flag.Usage()
		os.Exit(1)
	}

	cred, err := aws.EnvAuth()
	if err != nil {
		log.Fatalf("Couldn't auth into s3. Did you set up ENV? Error: %s", err)
	}
	s3client := s3.New(cred, aws.USWest)
	bucket := s3client.Bucket(*bucketName)

	// extract.
	checksum := fmt.Sprintf("%x", hashFiles(*toHash).Sum(nil))

	filename := *outfile
	// consider bucket.GetReader to pipe directly into gunzip/untar
	file, err := bucket.Get(fmt.Sprintf("%s/%s", checksum, filename))
	if err != nil {
		fmt.Printf("%s\n", err)
		build(*cmd)
		archive(*pkgDirs, filename)
		upload(bucket, checksum, filename)
	} else {
		extract(file)
	}
}
