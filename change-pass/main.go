package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/crypto/bcrypt"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type RequestBody struct {
	Pass        string `json:"pass"`
	ConfirmPass string `json:"confirm_pass"`
	Mail        string `json:"mail"`
	CodeUpdate  string `json:"code_update"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"msg,omitempty" bson:"msg"`
}

func returnArrayBytes(response Response) []byte {
	res, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return res
}

func returnApiGateway(r Response, st int) (events.APIGatewayProxyResponse, error) {
	resInBytes := returnArrayBytes(r)
	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "authorization, content-type",
		},
		Body:       fmt.Sprintf(string(resInBytes)),
		StatusCode: st,
	}, nil
}

func decodeTokenHeader(str string) (string, error) {
	bearer := strings.Split(str, " ")
	if len(bearer) < 2 {
		return "", errors.New("Authorization header malformed")
	}
	token := bearer[1]
	return token, nil
}

func getPass(pwd string) []byte {
	return []byte(pwd)
}

func hashAndSalt(pwd []byte) string {
	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}
	return string(hash)
}

func comparePasswords(hashedPwd string, plainPwd []byte) bool {
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "OPTIONS" {
		var response Response
		response.Success = true
		return returnApiGateway(response, 200)
	}

	var bodyData *RequestBody
	err := json.Unmarshal([]byte(request.Body), &bodyData)

	userDB, ok := os.LookupEnv("USER_DB")
	if !ok {
		userDB = "test"
	}
	pwdDB, ok := os.LookupEnv("PASSWORD_DB")
	if !ok {
		pwdDB = "xxxxxxxxxxxx"
	}
	hostDB, ok := os.LookupEnv("HOST_DB")
	if !ok {
		hostDB = "localhost"
	}
	portDB, ok := os.LookupEnv("PORT_DB")
	if !ok {
		portDB = "3306"
	}
	nameDB, ok := os.LookupEnv("NAME_DB")
	if !ok {
		nameDB = "test"
	}

	//connection
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v",
		userDB,
		pwdDB,
		hostDB,
		portDB,
		nameDB,
	))
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	psw1 := getPass(bodyData.Pass)
	transformPsw1 := hashAndSalt(psw1)
	psw2 := getPass(bodyData.ConfirmPass)

	check := comparePasswords(transformPsw1, psw2)
	if check != true {
		var res Response
		res.Success = false
		res.Message = "La confirmación de la contraseña no coincide."
		return returnApiGateway(res, 400)
	}

	update, err := db.Exec("UPDATE users SET pass = ?, code_update='' WHERE mail = ? and code_update = ?", transformPsw1, bodyData.Mail, bodyData.CodeUpdate)
	if err != nil {
		panic(err.Error())
	}

	row, err := update.RowsAffected()
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(row)
	fmt.Println(transformPsw1)

	if row < 1 {
		var res Response
		res.Success = false
		res.Message = "No se pudo actualizar su contraseña."
		return returnApiGateway(res, 200)
	}

	var response Response
	response.Success = true
	response.Message = "Contraseña actualizada"
	return returnApiGateway(response, 200)
}

func main() {
	lambda.Start(handler)
}
