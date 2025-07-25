package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func IsTgzFile(filename string) (bool, error) {
	// Vérifier l'extension
	if !hasTgzExtension(filename) {
		return false, nil
	}

	// Vérifier les magic bytes et la structure
	return validateTgzStructure(filename)
}

func ExtractTgz(src, dest string) error {
	// Ouvrir le fichier .tgz
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()

	// Créer un lecteur gzip
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("erreur création lecteur gzip: %v", err)
	}
	defer gzr.Close()

	// Créer un lecteur tar
	tr := tar.NewReader(gzr)

	// Créer le dossier de destination
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("erreur création dossier destination: %v", err)
	}

	// Extraire chaque fichier
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // Fin de l'archive
		}
		if err != nil {
			return fmt.Errorf("erreur lecture header tar: %v", err)
		}

		// Construire le chemin de destination de manière sécurisée
		target := filepath.Join(dest, header.Name)

		// Sécurité: vérifier que le chemin ne sort pas du dossier de destination
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("chemin non sécurisé: %s", header.Name)
		}

		// Traiter selon le type de fichier
		switch header.Typeflag {
		case tar.TypeDir:
			// Créer le dossier
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("erreur création dossier %s: %v", target, err)
			}

		case tar.TypeReg:
			// Créer le fichier
			if err := extractFile(tr, target, header); err != nil {
				return fmt.Errorf("erreur extraction fichier %s: %v", target, err)
			}

		case tar.TypeSymlink:
			// Créer le lien symbolique
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("erreur création symlink %s: %v", target, err)
			}

		default:
			fmt.Printf("Type de fichier non supporté: %c dans %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}

func extractFile(tr *tar.Reader, target string, header *tar.Header) error {
	// Créer les dossiers parents si nécessaire
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	// Créer le fichier
	f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
	if err != nil {
		return err
	}
	defer f.Close()

	// Copier le contenu
	if _, err := io.Copy(f, tr); err != nil {
		return err
	}

	return nil
}

func hasTgzExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".tgz" {
		return true
	}

	// Vérifier .tar.gz
	if ext == ".gz" {
		base := strings.TrimSuffix(filename, ext)
		return strings.ToLower(filepath.Ext(base)) == ".tar"
	}

	return false
}

func validateTgzStructure(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Vérifier les magic bytes gzip (1f 8b)
	magicBytes := make([]byte, 2)
	if _, err := file.Read(magicBytes); err != nil {
		return false, err
	}

	if magicBytes[0] != 0x1f || magicBytes[1] != 0x8b {
		return false, nil
	}

	// Revenir au début du fichier
	file.Seek(0, 0)

	// Essayer de créer un lecteur gzip
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return false, nil // Pas une erreur, juste pas un gzip valide
	}
	defer gzr.Close()

	// Essayer de créer un lecteur tar
	tr := tar.NewReader(gzr)

	// Essayer de lire au moins un header tar
	_, err = tr.Next()
	if err != nil {
		return false, nil // Pas un tar valide
	}

	return true, nil
}
