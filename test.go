package main

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
)

var (
	count = 0
	ss    = make(map[string]string)
	base  = "D:\\load\\konfi\\RELEASE_ZG7239G031AEZ34\\AppStdSwArchive\\Sharable"
)

func listfiles(dir string) []string {
	entries, err := os.ReadDir(path.Join(base, dir))
	if err != nil {
		panic(err)
	}
	files := make([]string, len(entries))
	for i, e := range entries {
		count++
		if e.IsDir() {
			_ = listfiles(path.Join(dir, e.Name()))
		} else {
			hash := sha1.New()
			f, err := os.Open(path.Join(base, dir, e.Name()))
			if err != nil {
				panic(err)
			}
			_, err = io.Copy(hash, f)
			if err != nil {
				panic(err)
			}
			sum := hex.EncodeToString(hash.Sum(nil))
			dstDir := path.Join("arc", sum[:2])
			if err := os.MkdirAll(dstDir, fs.ModeDir); err != nil {
				panic(err)
			}
			srcFileName := path.Join(dir, path.Base(f.Name()))
			srcFile, err := os.Open(path.Join(base, srcFileName))
			if err != nil {
				panic(err)
			}
			dstFileName := path.Join(dstDir, sum)
			_, err = os.Stat(dstFileName)
			if err == nil {
				log.Println("File exist", srcFileName, dstFileName)
			}
			dstFile, err := os.OpenFile(dstFileName, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				panic(err)
			}
			gz := gzip.NewWriter(dstFile)
			_, err = io.Copy(gz, srcFile)
			if err != nil {
				panic(err)
			}
			if err := gz.Flush(); err != nil {
				panic(err)
			}
			if err := gz.Close(); err != nil {
				panic(err)
			}
			if err := dstFile.Close(); err != nil {
				panic(err)
			}
			if err := srcFile.Close(); err != nil {
				panic(err)
			}
			log.Println(srcFileName, dstFileName)
			ss[srcFileName] = dstFileName
			// dst, err := os.OpenFile(path.Join("arc",))
			// gzip.NewWriter()
		}
		files[i] = e.Name()
	}
	return files
}
func mains() {
	_ = listfiles("")
	for dst, src := range ss {
		log.Println("Extract:", src, dst)
		srcFile, err := os.Open(src)
		if err != nil {
			panic(err)
		}
		dstFileName := path.Join("ext", dst)
		err = os.MkdirAll(path.Dir(dstFileName), fs.ModeDir)
		if err != nil {
			panic(err)
		}
		dstFile, err := os.OpenFile(dstFileName, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			panic(err)
		}
		gz, err := gzip.NewReader(srcFile)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(dstFile, gz)
		if err != nil {
			log.Println("zip error", err)
		}

	}
	log.Println(count)
}
