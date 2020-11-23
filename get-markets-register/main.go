package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type ResponseGeneral struct {
	Success bool  `json:"success"`
	Rows    []Row `json:"row" bson:"row"`
}

type Row struct {
	IDProduct  int    `json:"id_product"`
	IDCP       int    `json:"id_cp"`
	TypeMarket string `json:"type_market"`
	Name       string `json:"name"`
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

func returnArrayBytesGeneral(response ResponseGeneral) []byte {
	res, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return res
}

func returnApiGatewayGeneral(r ResponseGeneral, st int) (events.APIGatewayProxyResponse, error) {
	resInBytes := returnArrayBytesGeneral(r)
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "OPTIONS" {
		var response Response
		response.Success = true
		return returnApiGateway(response, 200)
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

	var wg sync.WaitGroup
	var response ResponseGeneral

	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := db.Query("SELECT id, id_postal_code, type_market, name FROM cat_markets where type_market = ? and active = 1 ORDER BY id ASC", request.PathParameters["type_market"])
		if err != nil {
			panic(err)
		}

		for results.Next() {
			var row Row
			err = results.Scan(&row.IDProduct, &row.IDCP, &row.TypeMarket, &row.Name)
			if err != nil {
				panic(err.Error())
			}
			response.Rows = append(response.Rows, row)
		}
	}()

	wg.Wait()

	if len(response.Rows) <= 0 {
		var res Response
		res.Success = false
		res.Message = "No se encontraron datos"
		return returnApiGateway(res, 200)
	}

	response.Success = true
	return returnApiGatewayGeneral(response, 200)
}

func main() {
	lambda.Start(handler)
}
