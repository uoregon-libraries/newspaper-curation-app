package fileutil

import (
	"fmt"
	"io"
	"os"
)

// CopyFile attempts to copy the bytes from src into dst, returning an error if
// applicable.  Does not use os.Link regardless of where the two files reside,
// as that can cause massive confusion when copying a file in order to back it
// up while writing out to the original.  The destination file permissions
// aren't set here, and must be managed externally.
func CopyFile(src, dst string) error {
	var err error
	var srcInfo os.FileInfo

	srcInfo, err = os.Stat(src)
	if err != nil {
		return fmt.Errorf("cannot stat %#v: %s", src, err)
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("cannot copy non-regular file %#v: %s", src, err)
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot stat %#v: %s", dst, err)
	}

	return copyFileContents(src, dst)
}

// copyFileContents actually copies bytes from src to dst.  On any error, an
// attempt is made to clean up the state of the filesystem (though this is not
// guaranteed) and the first error encountered is returned.  i.e., if there's a
// failure in the io.Copy call, the caller will get that error, not the
// potentially meaningless error in the call to close the destination file.
func copyFileContents(src, dst string) error {
	var srcFile, dstFile *os.File
	var err error

	// Open source file or exit
	srcFile, err = os.Open(src)
	if err != nil {
		return fmt.Errorf("unable to read %#v: %s", src, err)
	}
	defer srcFile.Close()

	// Create destination file or exit
	dstFile, err = os.Create(dst)
	if err != nil {
		return fmt.Errorf("unable to create %#v: %s", dst, err)
	}

	// Attempt to copy, and if the operation fails, attempt to clean up, then exit
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		err = fmt.Errorf("unable to copy data from %#v to %#v: %s", src, dst, err)
		dstFile.Close()
		os.Remove(dst)
		return err
	}

	// Attempt to sync the destination file
	err = dstFile.Sync()
	if err != nil {
		dstFile.Close()
		return fmt.Errorf("error syncing %#v: %s", dst, err)
	}

	// Attempt to close the destination file
	err = dstFile.Close()
	if err != nil {
		return fmt.Errorf("errro closing %#v: %s", dst, err)
	}

	return nil
}
