package services

import (
	"encoding/base64"
	"errors"
	"regexp"

	"github.com/paveltrufi/mantecabox/config"
	"github.com/paveltrufi/mantecabox/dao/factory"
	"github.com/paveltrufi/mantecabox/dao/interfaces"
	"github.com/paveltrufi/mantecabox/models"
	"github.com/paveltrufi/mantecabox/utilities/aes"
	"golang.org/x/crypto/bcrypt"
)

const (
	InvalidUsernameError = "invalid username (must be a valid nickname -all lowercases- with a length between 8 and 20 characters)"
	InvalidPasswordError = "password input is not SHA-512 hashed"
)

var (
	sha512Regex   *regexp.Regexp
	usernameRegex *regexp.Regexp
	userDao       interfaces.UserDao
)

func init() {
	dao := factory.UserDaoFactory(config.GetServerConf().Engine)
	userDao = dao

	sha512Compile, err := regexp.Compile(`^[A-Fa-f0-9]{128}$`)
	if err != nil {
		panic(err)
	}
	sha512Regex = sha512Compile
	usernameCompile, err := regexp.Compile(`(?i)^[a-z\d](?:[a-z\d]|_([a-z\d])){6,20}$`)
	if err != nil {
		panic(err)
	}
	usernameRegex = usernameCompile
}

func RegisterUser(c *models.Credentials) (models.User, error) {
	var user models.User
	if err := validateCredentials(c); err != nil {
		return user, err
	}
	decodedPassword, err := base64.URLEncoding.DecodeString(c.Password)
	if err != nil {
		return user, err
	}
	bcryptedPassword, err := bcrypt.GenerateFromPassword(decodedPassword, bcrypt.DefaultCost)
	if err != nil {
		return user, err
	}
	user.Credentials = models.Credentials{
		Username: c.Username,
		Password: base64.URLEncoding.EncodeToString(aes.Encrypt(bcryptedPassword)),
	}
	return userDao.Create(&user)
}

func validateCredentials(c *models.Credentials) error {
	decodedPassword, err := base64.URLEncoding.DecodeString(c.Password)
	if err != nil {
		return err
	}
	if matches := usernameRegex.MatchString(c.Username); !matches {
		return errors.New(InvalidUsernameError)
	}
	if matches := sha512Regex.Match(decodedPassword); !matches {
		return errors.New(InvalidPasswordError)
	}
	return nil
}
