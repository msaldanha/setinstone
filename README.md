
# SetInStone project

## What is it?

First, it is my pet project and isn't finished.

A P2P Social Network which uses a distributed DB to store posts.

It is called SetInStone because changes in the posts are not allowed and, once stored, can't be changed.

The posts form a chain (like a blockchain) and are cryptographically protected and tied to the previous post, making impossible to third parties tamper with them. 

The posts are stored in the IPFS network making them available everywhere in the world.

It is composed by tree packages:

- **anticorp**: the kernel, where all magic happens.
- **timeline**: implements a timeline (obvious :-) 
- **pulpit**: an app (api) that uses the timeline. You can post, like posts, reply/comment to posts.

## How to build?

Just run `go build .` on the root dir

## How to run?

On the root dir (after building it) run `./setinstone` without cmd line options and it will use default values:

```
  -data string
        Data Store file (default "8080.dat")
  -ipfsapiport string
        IPFS API port number (default "5002")
  -ipfsgatewayport string
        IPFS Gateway port number (default "8088")
  -ipfsport string
        IPFS port number (default "4001")
  -url string
        Listening address. Should have the form of [host]:port, i.e localhost:8080 or :8080 (default ":8080")
```

You can run another instance (to test things) just changing the values above to not cause conflicts.

## How to use?

NOTE: as setinstone is under development, it will set up IPFS node to use a temp directory. This means that the data will be discarded when the service stops. 

Assuming the service is running on localhost:8080, it will be waiting for requests on the configured host/port. All that is needed is to do http calls to the endpoints.

First, create a new address. This will create a new public/private key pair and store the private key on local the storage:

```
curl --location --request POST 'http://localhost:8080/addresses' \
--header 'Content-Type: application/json' \
--data-raw '{
	"password":"123456"
}'
```

Using the combination address/password, do a login to get a jwt token:

```
curl --location --request POST 'http://localhost:8080/login' \
--header 'Content-Type: application/json' \
--data-raw '{
	"address": "<INSERT HERE THE ADDRESS RETURNED FROM PREVIOUS CALL>",
	"password":"123456"
}'
```

Now, add a new post to a timeline using the received jwt:

```
curl --location --request POST 'http://localhost:8080/tl/pulpit/<INSERT HERE THE ADDRESS>' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <INSERT HERE THE JWT OBTAINED ON THE PREVIOUS CALL>' \
--data-raw '{
    "type": "Post",
    "postItem": {
        "mime_type": "plain/text",
        "data": "Message 1",
        "attachments": [
            "SOME-COOL-IMAGE.jpg"
        ],
        "connectors": [
            "like"
        ]
    }
}'
```

List all added posts:

```
curl --location --request GET 'http://localhost:8080/tl/pulpit/<INSERT HERE THE ADDRESS>?count=10' \
--header 'Authorization: Bearer <INSERT HERE THE JWT>'
```
