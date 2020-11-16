package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	uuid "github.com/satori/go.uuid"

	"net/smtp"

	"database/sql"

	"github.com/domodwyer/mailyak"

	_ "github.com/go-sql-driver/mysql"
)

type RequestBody struct {
	Mail string `json:"mail" bson:"mail"`
}

type Response struct {
	Success bool   `json:"success" bson:"success"`
	Message string `json:"msg" bson:"msg"`
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

func sendMail(to []string, code string) bool {
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
	mail.Subject("Cambio de contraseña")
	mail.HTML().Set("<section style='width: 600px; position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%); -webkit-transform: translate(-50%, -50%);'> <div style='background-color: #A9CE4D; padding: 18px; text-align: center; color: #fff; font-size: 18px; font-family: sans-serif; text-transform: uppercase;'>Cambio de contraseña</div> <div style='padding: 30px; font-family: sans-serif; font-size: 15px;'>Este correo ha sido enviado a " + to[0] + ", por favor da click aquí para poder cambiar tu contraseña: <a href='mimarchante.mx/nuevacontrasena.html?code=" + code + "&mail=" + to[0] + "'>Click aquí</a></div> <div style='font-size: 13px; padding: 30px; text-align: center; border-bottom: 1px solid #A9CE4D; border-left: 1px solid #A9CE4D; border-right: 1px solid #A9CE4D;'>Si no has solicitado este cambio, solamente borra este mensaje, saludos de parte del equipo mimarchante.mx</div></section>")
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
	if err != nil {
		panic(err.Error())
	}

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

	u1 := uuid.Must(uuid.NewV4())

	update, err := db.Exec("UPDATE users SET code_update = ? WHERE mail = ?", u1, bodyData.Mail)
	if err != nil {
		panic(err)
	}

	row, err := update.RowsAffected()
	if err != nil {
		panic(err)
	}

	if row > 0 {
		okSend := sendMail([]string{bodyData.Mail}, u1.String())

		if okSend {
			var res Response
			res.Success = true
			res.Message = "Mensaje enviado."
			return returnApiGateway(res, 200)
		}
	}

	//okSend := sendMail([]string{bodyData.Mail})

	//if !okSend {
	var res Response
	res.Success = false
	res.Message = "Error al enviar el mensaje"
	return returnApiGateway(res, 400)
	//}

	/*var res Response
	res.Success = true
	res.Message = "Mensaje enviado."
	return returnApiGateway(res, 200)*/
}

func main() {
	lambda.Start(handler)
}
