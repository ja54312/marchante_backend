package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
)

type RequestBody struct {
	IDUser       int     `json:"id_user" bson:"id_user"`
	IDRol        int     `json:"id_roll" bson:"id_roll"`
	CategoryType int     `json:"id_type_category" bson:"id_type_category"`
	Name         string  `json:"name" bson:"name"`
	PricePZ      float32 `json:"price_pz" bson:"price_pz"`
	PriceKG      float32 `json:"price_kg" bson:"price_kg"`
}

type Response struct {
	Success bool   `json:"success" bson:"success"`
	Message string `json:"msg" bson:"msg"`
}

type Product struct {
	IDProduct int     `json:"id_product" bson:"id_product"`
	Name      string  `json:"name" bson:"name"`
	Type      string  `json:"type" bson:"type"`
	Count     int     `json:"count" bson:"count"`
	Price     float32 `json:"price" bson:"price"`
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

func checkToken(token string) error {
	jwtKey, ok := os.LookupEnv("KEY_JWT")
	if !ok {
		fmt.Println("no existe key para jwt")
		os.Exit(1)
	}
	tok, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})
	if tok.Valid {
		fmt.Println("all ok")
		return nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			fmt.Println("Malformed Token")
			return err
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			fmt.Println("Token Expired")
			return err
		} else {
			fmt.Println("Could not handled this token", err)
			return err
		}
		//return err
	} else {
		fmt.Println("Could not handled this token", err)
		return err
	}
}

func getAndCheckToken(request events.APIGatewayProxyRequest) error {
	auth, ok := request.Headers["Authorization"]
	if !ok {
		//log.Println("La cabecera no presenta una Autorización")
		//os.Exit(1)
		err := errors.New("not header exist such as Bearer token")
		return err
	}

	token, erro := decodeTokenHeader(auth)
	if erro != nil {
		log.Println(erro)
		return erro
	}

	erro = checkToken(token)
	if erro != nil {
		return erro
	}

	return nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "OPTIONS" {
		var response Response
		response.Success = true
		return returnApiGateway(response, 200)
	}

	er := getAndCheckToken(request)
	if er != nil {
		var response Response
		response.Success = false
		response.Message = "Se requiere autenticación y/o token invalido"
		return returnApiGateway(response, 401)
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
	datetime := currentTime.Format("2006-01-02 15:04:05")

	insert, err := db.Exec("INSERT INTO products (id_user, id_rol, id_type_category, name, price_pz, price_kg, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", bodyData.IDUser, bodyData.IDRol, bodyData.CategoryType, bodyData.Name, bodyData.PricePZ, bodyData.PriceKG, datetime)
	if err != nil {
		fmt.Println("error al insertar: ", err.Error())
	}

	affected, _ := insert.RowsAffected()

	if affected <= 0 {
		var res Response
		res.Success = false
		res.Message = "Los datos no se han podido guardar"
		return returnApiGateway(res, 400)
	}

	var res Response
	res.Success = true
	res.Message = "Registro guardado exitosamente"
	return returnApiGateway(res, 200)
}

func main() {
	lambda.Start(handler)
}
