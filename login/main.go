package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"database/sql"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
)

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type Body struct {
	Active int    `json:"active"`
	Pass   string `json:"pass"`
}

type Response struct {
	Success bool     `json:"success" bson:"success"`
	Message string   `json:"msg,omitempty" bson:"msg"`
	Token   string   `json:"token,omitempty" bson:"token"`
	Data    DataUser `json:"data_user,omitempty" bson:"data_user"`
}

type DataUser struct {
	IDUser       int    `json:"id_user,omitempty" bson:"id_user"`
	Customer     int    `json:"id_rol,omitempty" bson:"id_rol"`
	TypeMarket   string `json:"type_market,omitempty" bson:"type_market"`
	Zone         string `json:"zone,omitempty" bson:"zone"`
	Market       string `json:"market,omitempty" bson:"market"`
	Local        string `json:"local,omitempty" bson:"local"`
	NameUser     string `json:"name_user,omitempty" bson:"name_user"`
	Mail         string `json:"mail,omitempty" bson:"mail"`
	NameTypeUser string `json:"name_type_user,omitempty" bson:"name_type_user"`
	Redirect     string `json:"redirect,omitempty" bson:"redirect"`
}

type RequestBody struct {
	Mail string `json:"mail" bson:"mail"`
}

type ResponseSuccess struct {
	Cellphone   int64  `json:"cellphone" bson:"cellphone"`
	ImagePerfil string `json:"image_perfil" bson:"image_perfil"`
	Mail        string `json:"mail" bson:"mail"`
	Name        string `json:"name" bson:"name"`
	Pass        string `json:"pass,omitempty" bson:"pass,omitempty"`
}

func getToken() (string, error) {
	jwtKey, ok := os.LookupEnv("KEY_JWT")
	if !ok {
		fmt.Println("no existe key para jwt")
	}
	expirationTime := time.Now().Add(10 * time.Hour)
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Issuer:    "user",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signToken, err := token.SignedString([]byte(jwtKey))
	return signToken, err
}

func checkToken(token string) error {
	jwtKey, ok := os.LookupEnv("KEY_JWT")
	if !ok {
		fmt.Println("no existe key para jwt")
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
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			fmt.Println("Token Expired")
		} else {
			fmt.Println("Could not handled this token", err)
		}
		return err
	} else {
		fmt.Println("Could not handled this token", err)
		return err
	}
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

func comparePasswords(hashedPwd string, plainPwd []byte) bool {
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func returnApiGateway(r Response, st int) (events.APIGatewayProxyResponse, error) {
	resInBytes := returnArrayBytes(r)
	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST, GET, OPTIONS, HEAD",
			"Access-Control-Allow-Headers": "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
		},
		Body:       fmt.Sprintf(string(resInBytes)),
		StatusCode: st,
	}, nil
}

func returnCustomApiGateway(res ResponseSuccess, st int) (events.APIGatewayProxyResponse, error) {
	resInBytes := returnCustomArrayBytes(res)
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf(string(resInBytes)),
		StatusCode: st,
	}, nil
}

func returnCustomArrayBytes(response ResponseSuccess) []byte {
	res, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return res
}

func decodeUserPass(str string) (string, string, error) {
	b64split := strings.Split(str, " ")
	if len(b64split) < 2 {
		return "", "", errors.New("Authorization header malformed")
	}
	b64pass := b64split[1]
	decoded, err := base64.StdEncoding.DecodeString(b64pass)
	if err != nil {
		return "", "", err
	}
	spldecoded := strings.Split(string(decoded), ":")
	if len(spldecoded) <= 1 {
		return "", "", errors.New("User or Password Incorrect")
	}
	if len(spldecoded) > 2 {
		username := spldecoded[0]
		pass := strings.Join(spldecoded[1:], "")
		return username, pass, nil
	}
	return spldecoded[0], spldecoded[1], nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "OPTIONS" {
		var res Response
		res.Success = true
		return returnApiGateway(res, 200)
	}
	auth, ok := request.Headers["Authorization"] //get header data, user and pass
	if !ok {
		log.Println("La cabecera no presenta una Autorización")
		var res Response
		res.Success = false
		res.Message = "La cabecera no presenta una Autorización"
		return returnApiGateway(res, 400)
	}

	user, pass, err := decodeUserPass(auth)
	if err != nil {
		log.Println(err)
		var res Response
		res.Success = false
		res.Message = "Se requiere autenticación"
		return returnApiGateway(res, 401)
	}

	//genero el token
	token, err := getToken()
	if err != nil {
		log.Println(err)
		var res Response
		res.Success = false
		res.Message = "No se pudo firmar el token"
		return returnApiGateway(res, 401)
	}

	/*fmt.Println("datos user")
	fmt.Println(user)
	fmt.Println(pass)*/
	err = checkToken(token)
	if err != nil {
		var res Response
		res.Success = false
		res.Message = "Token invalido"
		returnApiGateway(res, 401)
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

	results, err := db.Query("SELECT active, pass FROM users WHERE mail = ? and active = 1", user)
	if err != nil {
		var res Response
		res.Success = false
		res.Message = "No existen datos para esta consulta"
		return returnApiGateway(res, 400)
	}

	var row Body
	for results.Next() {
		err = results.Scan(&row.Active, &row.Pass)
		if err != nil {
			panic(err.Error())
		}
	}

	if row.Active <= 0 {
		var res Response
		res.Success = false
		res.Message = "No existen datos para esta consulta"
		return returnApiGateway(res, 200)
	}

	psw := getPass(pass) //transform to array bytes
	check := comparePasswords(row.Pass, psw)

	if check != true {
		var res Response
		res.Success = false
		res.Message = "La contraseña no es correcta"
		return returnApiGateway(res, 400)
	}

	resultsData, err := db.Query("SELECT a.id, a.id_type_user, a.type_market, a.zone, a.market, a.local, a.name, a.mail, b.nombre, b.redirect FROM users as a left join type_users as b on a.id_type_user = b.id WHERE a.mail = ? and a.active = 1", user)
	if err != nil {
		var response Response
		response.Success = false
		response.Message = "No existen datos para esta consulta"
		return returnApiGateway(response, 400)
	}

	var res Response
	for resultsData.Next() {
		err = resultsData.Scan(&res.Data.IDUser, &res.Data.Customer, &res.Data.TypeMarket, &res.Data.Zone, &res.Data.Market, &res.Data.Local, &res.Data.NameUser, &res.Data.Mail, &res.Data.NameTypeUser, &res.Data.Redirect)
		if err != nil {
			panic(err.Error())
		}
	}

	if res.Data.IDUser <= 0 {
		var response Response
		response.Success = false
		response.Message = "No existen datos para esta consulta"
		return returnApiGateway(response, 400)
	}

	res.Success = true
	res.Token = token

	return returnApiGateway(res, 200)
}

func main() {
	lambda.Start(handler)
}
