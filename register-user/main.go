package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type RequestBody struct {
	Customer   string `json:"customer" bson:"customer"`
	TypeMarket string `json:"type_market" bson:"type_market"`
	Zone       string `json:"zone" bson:"zone"`
	Market     string `json:"market" bson:"market"`
	Local      string `json:"local" bson:"local"`
	Name       string `json:"name" bson:"name"`
	Mail       string `json:"mail" bson:"mail"`
	Pass       string `json:"pass" bson:"pass"`
	CreatedAt  string `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

type RequestGet struct {
	Cellphone int64 `json:"cellphone" bson:"cellphone"`
}

type ResponseVerifyMail struct {
	ID int `json:"id" bson:"id"`
}

type Response struct {
	Success bool   `json:"success" bson:"success"`
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

/*func comparePasswords(hashedPwd string, plainPwd []byte) bool {
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}*/

func returnApiGateway(r Response, st int) (events.APIGatewayProxyResponse, error) {
	resInBytes := returnArrayBytes(r)
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf(string(resInBytes)),
		StatusCode: st,
	}, nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	currentTime := time.Now()
	bodyData.CreatedAt = currentTime.Format("2006-01-02 15:04:05")

	psw1 := getPass(bodyData.Pass)
	transformPsw1 := hashAndSalt(psw1)
	bodyData.Pass = transformPsw1

	var response Response

	results, err := db.Query("SELECT id FROM users WHERE mail = ?", bodyData.Mail)
	if err != nil {
		fmt.Println(err.Error())
	}

	var response_verify ResponseVerifyMail
	for results.Next() {
		err = results.Scan(&response_verify.ID)
		if err != nil {
			panic(err.Error())
		}
	}

	if response_verify.ID > 0 {
		response.Success = false
		response.Message = "El usuario " + bodyData.Mail + " ya ha sido registrado."
		return returnApiGateway(response, 400)
	}

	insert, err := db.Exec("INSERT INTO users (customer, type_market, zone, market, local, name, mail, pass, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", bodyData.Customer, bodyData.TypeMarket, bodyData.Zone, bodyData.Market, bodyData.Local, bodyData.Name, bodyData.Mail, bodyData.Pass, bodyData.CreatedAt)
	if err != nil {
		fmt.Println(err.Error())
	}

	if err != nil {
		response = Response{false, "Error al guardar los datos, intentelo nuevamente."}
		return returnApiGateway(response, 400)
	}

	check_insert, _ := insert.RowsAffected()
	if check_insert > 0 {
		response = Response{true, "Los datos se han guardado correctamente."}
	}

	return returnApiGateway(response, 200)
}

func main() {
	lambda.Start(handler)
}
