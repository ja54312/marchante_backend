package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/domodwyer/mailyak"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type RequestBody struct {
	Customer   string `json:"customer" bson:"customer"`
	TypeMarket string `json:"type_market" bson:"type_market"`
	Zone       string `json:"zone" bson:"zone"`
	Market     int    `json:"id_market" bson:"id_market"`
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

func sendMail(to []string, typeUser string) bool {
	host, ok := os.LookupEnv("HOST_MAIL")
	if !ok {
		return false
	}
	user, ok := os.LookupEnv("USER_MAIL")
	if !ok {
		return false
	}
	pwd, ok := os.LookupEnv("PASS_MAIL")
	if !ok {
		return false
	}

	auth := smtp.PlainAuth("", user, pwd, host)

	mail := mailyak.New(host+":587", auth)

	mail.To(to...)
	mail.From(user)
	mail.FromName("Mi Marchante - Unidos por México")
	mail.Subject("Registro de usuario")
	mail.HTML().Set("<section style='width: 600px; position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%); -webkit-transform: translate(-50%, -50%);'> <div style='background-color: #A9CE4D; padding: 18px; text-align: center; color: #fff; font-size: 18px; font-family: sans-serif; text-transform: uppercase;'>Registro de usuario</div> <div style='padding: 30px; font-family: sans-serif; font-size: 15px;'>Se ha creado el usuario con el correo: " + to[0] + ", el tipo de usuario que has creado es:  " + typeUser + ", bienvenido a Mi Marchante para poder ingresar entre <a href='https://mimarchante.mx'>aquí</a></div> <div style='font-size: 13px; padding: 30px; text-align: center; border-bottom: 1px solid #A9CE4D; border-left: 1px solid #A9CE4D; border-right: 1px solid #A9CE4D;'>Saludos de parte del equipo MiMarchanteMX</div></section>")
	if err := mail.Send(); err != nil {
		return false
	}
	fmt.Println("enviado")
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

	insert, err := db.Exec("INSERT INTO users (id_type_user, type_market, zone, id_market, local, name, mail, pass, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", bodyData.Customer, bodyData.TypeMarket, bodyData.Zone, bodyData.Market, bodyData.Local, bodyData.Name, bodyData.Mail, bodyData.Pass, bodyData.CreatedAt)
	if err != nil {
		fmt.Println(err.Error())
	}

	if err != nil {
		response = Response{false, "Error al guardar los datos, intentelo nuevamente."}
		return returnApiGateway(response, 400)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var customer string
		switch bodyData.Customer {
		case "1":
			customer = "Locatario"
		case "2":
			customer = "Cliente"
		case "3":
			customer = "Admin"
		}
		ok := sendMail([]string{bodyData.Mail}, customer)
		fmt.Println(ok)
	}()

	wg.Wait()

	check_insert, _ := insert.RowsAffected()
	if check_insert > 0 {
		response = Response{true, "Los datos se han guardado correctamente."}
	}

	return returnApiGateway(response, 200)
}

func main() {
	lambda.Start(handler)
}
