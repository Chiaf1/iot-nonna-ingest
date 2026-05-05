package utils

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/natefinch/atomic"
)

// WriteFileAtomic saves the file "path" using an atomic sequence.
// First it creates the path if not existing it creates it then calls the write file atomic.
// first it saves the "data" to a tempFile, then it updates the permissions
// of the tempFile to "perm" and then it replaces the original file "path" with the tempFile. This is usefull for files that are core to the app
// and can't risk to be corrupted.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	//I get the directory from the path
	dir := filepath.Dir(path)

	//create the directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Cannot create directory: %w", err)
	}

	// create an io.reader for atomic.WriteFile
	reader := bytes.NewReader(data)

	//atomicaly writing the file
	if err := atomic.WriteFile(path, reader); err != nil {
		return fmt.Errorf("Atomic write failed: %w", err)
	}

	//set permissions
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("Error setting permissions: %w", err)
	}

	/*
			questa era la mia versione di scrittura file atomica. Funziona correttamente solo su Linux. Su windows dovevo implementare
			una funzione diversa. ma atomic.WirteFile già fa queste cose quindi non mi serve crearla a mano.

		//I Create the temp file used to save the data
		tmp, err := os.CreateTemp(dir, "atomic-*")
		if err != nil {
			return fmt.Errorf("Error creating the temp file: %w", err)
		}
		tmpName := tmp.Name()

		//write data to the temp file
		if _, err = tmp.Write(data); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			return fmt.Errorf("Error writing data to temp file: %w", err)
		}

		//flush the disck
		if err = tmp.Sync(); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			return fmt.Errorf("Error syncing temp file: %w", err)
		}

		//closing the file
		if err = tmp.Close(); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("Error closing temp file: %w", err)
		}

		//set permissions
		if err = os.Chmod(tmpName, perm); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("Error setting permissions: %w", err)
		}

		//atomic rename of the file
		if err = os.Rename(tmpName, path); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("Erorr replacing file: %w", err)
		}
	*/
	return nil
}
