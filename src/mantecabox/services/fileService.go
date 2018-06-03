package services

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"mantecabox/dao/factory"
	"mantecabox/dao/interfaces"
	"mantecabox/models"
	"mantecabox/utilities"

	"github.com/go-http-utils/headers"
	"github.com/sirupsen/logrus"
)

const userReadablePerms = 0600

type (
	FileService interface {
		GetAllFiles(user models.User) ([]models.File, error)
		CreateFile(file *models.File) (models.File, error)
		DeleteFile(file int64, fileID string) error
		GetFile(filename string, user *models.User) (models.File, error)
		UpdateFile(id int64, file models.File) (models.File, error)
		SaveFile(file multipart.File, uploadedFile models.File) error
		GetDecryptedLocalFile(file models.File) ([]byte, error)
		GetFileStream(fileDecrypt []byte, file models.File) (contentLength int64, contentType string, reader *bytes.Reader, extraHeaders map[string]string)
		createDirIfNotExist()
	}

	FileServiceImpl struct {
		configuration *models.Configuration
		fileDao       interfaces.FileDao
		aesCipher     utilities.AesCTRCipher
	}
)

func NewFileService(configuration *models.Configuration) FileService {
	if configuration == nil {
		return nil
	}
	if configuration.FilesPath == "" {
		configuration.FilesPath = "files"
	}
	// Maybe the config path didn't ended with folder's slash, so we add it
	if configuration.FilesPath[len(configuration.FilesPath)-1] != '/' {
		configuration.FilesPath += "/"
	}
	fileServiceImpl := FileServiceImpl{
		configuration: configuration,
		fileDao:       factory.FileDaoFactory(configuration.Database.Engine),
		aesCipher:     utilities.NewAesCTRCipher(configuration.AesKey),
	}
	fileServiceImpl.createDirIfNotExist()
	return fileServiceImpl
}

func (fileService FileServiceImpl) GetAllFiles(user models.User) ([]models.File, error) {
	return fileService.fileDao.GetAll(&user)
}

func (fileService FileServiceImpl) CreateFile(file *models.File) (models.File, error) {
	return fileService.fileDao.Create(file)
}

func (fileService FileServiceImpl) DeleteFile(file int64, fileID string) error {

	err := fileService.fileDao.Delete(file)
	if err != nil {
		return err
	}

	err = os.Remove(fileService.configuration.FilesPath + fileID)
	if err != nil {
		return err
	}

	return err
}

func (fileService FileServiceImpl) GetFile(filename string, user *models.User) (models.File, error) {
	return fileService.fileDao.GetByPk(filename, user)
}

func (fileService FileServiceImpl) UpdateFile(id int64, file models.File) (models.File, error) {
	return fileService.fileDao.Update(id, &file)
}

func (fileService FileServiceImpl) SaveFile(file multipart.File, uploadedFile models.File) error {
	// Conversión a bytes del fichero
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return err
	}
	encrypted := fileService.aesCipher.Encrypt(buf.Bytes())
	// Guardamos el fichero encriptado
	if err := ioutil.WriteFile(fileService.configuration.FilesPath+string(uploadedFile.Id), encrypted, userReadablePerms); err != nil {
		return err
	}
	return nil
}

func (fileService FileServiceImpl) GetDecryptedLocalFile(file models.File) ([]byte, error) {
	fileEncrypt, err := ioutil.ReadFile(fileService.configuration.FilesPath + string(file.Id))
	if err != nil {
		return nil, err
	}

	return fileService.aesCipher.Decrypt(fileEncrypt), err
}

func (fileService FileServiceImpl) GetFileStream(fileDecrypt []byte, file models.File) (contentLength int64, contentType string, reader *bytes.Reader, extraHeaders map[string]string) {
	reader = bytes.NewReader(fileDecrypt)
	contentLength = reader.Size()
	contentType = http.DetectContentType(fileDecrypt)

	extraHeaders = map[string]string{
		headers.ContentDisposition: `attachment; filename="` + file.Name + `"`,
	}

	return
}

func (fileService FileServiceImpl) createDirIfNotExist() {
	err := os.MkdirAll(fileService.configuration.FilesPath, userReadablePerms)
	if err != nil {
		logrus.Print("Error creating file's directory: " + err.Error())
		panic(err)
	}
}
