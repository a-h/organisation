run-dynamo:
	docker run -p 8000:8000 -v `pwd`/dbstore:/dbstore amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb -dbPath /dbstore

