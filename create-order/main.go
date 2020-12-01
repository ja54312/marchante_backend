package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
)

type RequestBody struct {
	IDUser   int           `json:"id_user" bson:"id_user"`
	IDMarket int           `json:"id_market" bson:"id_market"`
	Cart     []CartProduct `json:"cart" bson:"cart"`
}

type Response struct {
	Success bool   `json:"success" bson:"success"`
	IDOrder int64  `json:"id_order,omitempty" bson:"id_order"`
	Message string `json:"msg" bson:"msg"`
}

type CartProduct struct {
	IDTenant  int     `json:"id_tenant" bson:"id_tenant"`
	IDProduct int     `json:"id_product" bson:"id_product"`
	Quantity  float32 `json:"quantity" bson:"quantity"`
	PricePZ   float32 `json:"price_pz" bson:"price_pz"`
	PriceKG   float32 `json:"price_kg" bson:"price_kg"`
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

func CurrentTimeMex(t time.Time, name string) (time.Time, error) {
	loc, err := time.LoadLocation(name)
	if err == nil {
		t = t.In(loc)
	}
	return t, err
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

	var wg sync.WaitGroup
	currentTime, _ := CurrentTimeMex(time.Now(), "America/Mexico_City")
	datetime := currentTime.Format("2006-01-02 15:04:05")
	var totalOrder float32

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range bodyData.Cart {
			var subtotal float32

			if bodyData.Cart[i].PricePZ > 0 {
				subtotal = bodyData.Cart[i].PricePZ * bodyData.Cart[i].Quantity
			}
			if bodyData.Cart[i].PriceKG > 0 {
				subtotal = bodyData.Cart[i].PriceKG * bodyData.Cart[i].Quantity
			}

			totalOrder += subtotal
		}
	}()
	wg.Wait()

	insert, err := db.Exec("INSERT INTO orders (id_user, id_market, total, created_at) VALUES (?, ?, ?, ?)", bodyData.IDUser, bodyData.IDMarket, totalOrder, datetime)
	if err != nil {
		fmt.Println("error al insertar: ", err.Error())
	}

	last, _ := insert.LastInsertId()

	if last <= 0 {
		var res Response
		res.Success = false
		res.Message = "La orden no se ha podido guardar"
		return returnApiGateway(res, 400)
	}

	for i := range bodyData.Cart {
		wg.Add(1)

		go func() {
			defer wg.Done()
			var total float32

			if bodyData.Cart[i].PricePZ > 0 {
				total = bodyData.Cart[i].PricePZ * bodyData.Cart[i].Quantity
			}
			if bodyData.Cart[i].PriceKG > 0 {
				total = bodyData.Cart[i].PriceKG * bodyData.Cart[i].Quantity
			}
			insertDetail, err := db.Exec("INSERT INTO detail_order (id_order, id_tenant, id_product, quantity, price_pz, price_kg, total) VALUES (?, ?, ?, ?, ?, ?, ?)", last, bodyData.Cart[i].IDTenant, bodyData.Cart[i].IDProduct, bodyData.Cart[i].Quantity, bodyData.Cart[i].PricePZ, bodyData.Cart[i].PriceKG, total)
			if err != nil {
				fmt.Println("error al insertar: ", err.Error())
			}
			lastIDDetail, _ := insertDetail.LastInsertId()
			if lastIDDetail <= 0 {
				fmt.Println("error al insertar el detalle de orden ", last)
			}
		}()

		wg.Wait()
	}

	var res Response
	res.Success = true
	res.IDOrder = last
	res.Message = "Orden registrada con éxito"
	return returnApiGateway(res, 200)
}

func main() {
	lambda.Start(handler)
}
