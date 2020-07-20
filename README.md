# Rental-Management
The rental management repo holds all code for the rental management application. 

# Sensor Decoding
Senet folder is the interface used to decode messages received from the Senet network. It will decode the standard Senet Packet.
It will also decode the Senet Payload depending on sensor manufacturer and type.

# AWS Lambda and Dynamo DB
senetLamba is a lambda function that utilizes the senet decoding interface to decode messages that are forwarded to the AWS API endpoint.
After decoding, the data is then written to a AWS DynamoDB.
