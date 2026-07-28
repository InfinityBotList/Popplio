package assetmanager

import (
	"errors"
	"io/fs"
	"os"
)

// Deletes a file if it exists
func DeleteFileIfExists(path string) error {
	st, err := os.Stat(path)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return err
	}

	if st.IsDir() {
		return errors.New("path is a directory")
	}

	// Delete file
	err = os.Remove(path)

	if err != nil {
		return err
	}

	return nil
}
