package main

import (
	"context"
	"fmt"
	"net/http"
	"senet"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

//SenetEvent is the struct for saving the json object from the event request
type SenetEvent struct {
	Body string `json:"body"`
}

type SensorDataDBItem struct {
	SensorId     string
	Timestamp    string
	Data         map[string]string
	HasAlert     bool
	AlertCleared bool
}

//HandleRequest is the AWS Lambda handler
func HandleRequest(ctx context.Context, req SenetEvent) (http.Response, error) {
	response := http.Response{
		Status:     "200 Ok",
		StatusCode: 200,
	}
	fmt.Println("Starting request")
	//get request data and return json object to put into db
	body := req.Body
	packetStruct := senet.DecodeSenetPacket(body)

	devID := packetStruct.GetDevEUI()
	pdu := packetStruct.GetPdu()
	// TODO: use db to search devid and get manufacturer and type
	fmt.Println("DevID: " + devID + " PDU: " + pdu)
	sensor := senet.New("RadioBridge", "Temperature", pdu)

	//put into db
	//Create item and marshal it for AttributeValue
	dbItem := SensorDataDBItem{
		SensorId:     devID,
		Timestamp:    packetStruct.Txtime,
		Data:         sensor.GetData(),
		HasAlert:     sensor.HasAlert(),
		AlertCleared: !sensor.HasAlert(),
	}
	fmt.Println("Creating DB item")
	fmt.Println(dbItem)
	av, err := dynamodbattribute.MarshalMap(dbItem)
	if err != nil {
		panic(fmt.Sprintf("failed to DynamoDB marshal dbItem, %v", err))
	}

	svc := dynamodb.New(session.New())
	input := &dynamodb.PutItemInput{
		TableName: aws.String("SensorData"),
		Item:      av,
	}

	result, err := svc.PutItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				fmt.Println(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeTransactionConflictException:
				fmt.Println(dynamodb.ErrCodeTransactionConflictException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return http.Response{Status: "500 Internal Error", StatusCode: 500}, err
	}
	fmt.Println(result)
	return response, nil
}

func main() {

	lambda.Start(HandleRequest)
}
