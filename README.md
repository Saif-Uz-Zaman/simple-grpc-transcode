# Simple gRPC Transcoding
---

## Setup

### 1. Prepare Enviornemnt
Configure and run minikube
```
$ minikube config set cpus 4 
$ minikube config set memory 12384
$ minikube start
```

In this demo we are going to use `MetalLB` for loadbalancer provider. Install and configure metallb with a ip range (in my case, its 192.168.49.105-192.168.49.120)
```
$ minikube addons enable metallb
$ minikube addons configure metallb
```

In this demo we are also going to use `istio`. Install `istio` and load default profile.
```
$ istioctl install
```
### 2. Protos and anotate gRPC services Using google api
We must annotate the RPC services that we would want to expose as a REST endpoint. First we need to include `google/api/annotations.proto`. So that we can able to generate `protocol descriptor sets` which are needed to mount in the envoy configs.

```
$ git clone https://github.com/googleapis/googleapis googleapis-proto
```
Now we need to specify `GOOGLEAPIS_DIR` variable with the `google/api/annotations.proto` directory in our `makefile`

### 3. Generating protocol descriptor sets
We can now able to generate `protocol descriptor sets`.

```
$ make proto-all
```
This command will generate all necessary files for the application related to the proto, and the configmap yamls in the respected directories for both services `user` & `transaction`.

We need the `protocol descriptor sets` mounted on the envoy proxy. So we pass the `protocol descriptor sets` as a data in a configmap

`make proto-all` generate the files for both services. You can also run them individually,
```
$ make proto-user
$ make proto-transaction
```

### 4. Generating key pairs for signing the jwt token

We need public private key pair for siging the jwt token from the `user` service using `private-key` and verify the signed jwt token with `public-key`

```
$ mkdir keys
$ openssl genrsa -out keys/priv.key 2048
$ openssl rsa -in keys/priv.key -pubout -out keys/pub.key
```
Though we should not let the key passing with a codebases or with the apllication, it should be mounted as a secret in kubernets. But for simplicty in this demo I will pass the keys along with the user apllication in the docker container.

---
## TODO:
Pass keys with kubernetes secret and mount it in the respective application pod.

---

### 5. Build Applications
In this demo we are going to use build the aplications and pass the necessary binarries to the respective `container` folder. So that we don't need to rebuild the applications again during `docker-build`.

As we decided moving with the `keys in the container image` we need to also pass the keys un the repective `container` folder.

We can do this using,

```
// for all applications at once
$ make build-all

// or,

// for individual applications
$ make build-user
$ make build-transaction
```

### 6. Build & Push Docker Images and Update K8S Manifests

In this step we are going to build and push docker images and update respective kubernetes manifests with updated ImageName and ImageTag

- Change `DOCKER_REPO` and `RELEASE` variables with the appropriate values in the `makefile`
- Create two repositories in your docker registry, named `${APP_PREFIX}-user`, `${APP_PREFIX}-transaction`

Now, we can do,

```
// for all applications at once
$ make docker-all

// or,

// for individual applications
$ make docker-user
$ make docker-transaction
```

### 7. Deploy Applications in K8S

#### 7.1 Create Namespace

- Create namespace and label it for istio injection

```
$ kubectl create ns dev
$ kubectl label ns dev istio-injection=enabled                                                                     
```

#### 7.2 Deploy Postgres

- Deploy a `demo` postgres instance with two databases for two services.
- Check `manifests/postgres.yaml`
- This database is for demo purpose
- In production we should use `statefulSets`, `replicas`, `persistent volume` and `PodDisruptionBudget`

```
$ make deploy-pg
```

#### 7.3 Deploy Applications

We don't need to change any kubernetes manifests for this is steps, like image or. tag update. It is done in the previous step.

```
$ make deploy-user
$ make deploy-transaction
```

#### 7.4 Deploy Gateway & VirtualService

Here is our api call flow,

![Traffic Flow](assets/gateway.png "Traffic Flow")

- Feel free to check the manifests in the `manifests/gateway` folder
- Feel free to change the `hosts` section for both Gateway & VirtualService manifest files
- The default `host` is `www.simple-grpc-transcode.com`
- Point the dns entry to the respective loadbalancer ip. In this case we use hostentry
- You can get the loadBalancer IP using
`kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`


** In case you want to do the step from 3 to 7 you can just use `make all`


## Demo

1. Let's seed some users with balance

    ```
    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/seeduser -d \
    '{ "name": "foo","balance": "1000"}'

    // Output:
    // {
    //  "id": 1
    // }

    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/seeduser -d \
    '{ "name": "bar","balance": "1500"}'

    // Output:
    // {
    //  "id": 2
    // }

    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/seeduser -d \
    '{ "name": "bazz","balance": "2000"}'

    // Output:
    // {
    //  "id": 3
    // }

    ```

2. Login for users and get the `signed jwt token` as response body

    ```
    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/login -d '{ "id": "1" }'

    // Output:
    // {
    //  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VyTmFtZSI6ImZvbyIsIklkIjoxLCJpc3MiOiJzaW1wbGUtand0LXByb3ZpZGVyIiwic3ViIjoic2ltcGxlLWp3dC1wcm92aWRlciIsImV4cCI6MTY2NDE0OTI5NX0.Q5_39FYbNtJRaHy7tpBlg1pzHHfjAo7r6n8tX-UKk35UVWQhkUTo6DwrJ7jbmcez4A-jwvM0g9LU5mOgdGYbr7S9iBk6j6Tb7GMhYHuXad2o_q_5eZs_NjB_Ei3mq6Xr3SqOi8wbaCojyywN-HgzPw6tiAgq5Ow9UYAVXaFw-UqIfKKHhkD0ro2KPU-yo9w_N-yRTSxAvp0FxP6XUAw6oA6tRzHwAvRm4A8mcFKw_5645dczhRi1JflOthkcQA2SoXxtBEHWJMP8vchUbSuWSsNL7P_RoHctWiO7_5gAc_hVoW2HYioKB01-xtZBm942e81-uIADRPzeFqg39krz3g"
    // }

    ```

3. Get the user balance

    ```
    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/balance -d '{ "id": "1" }'

    // Output:
    // {
    //  "balance": 1000
    // }

    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/balance -d '{ "id": "3" }'

    // Output:
    // {
    //  "balance": 2000
    // }

    ```

    After those operations userdb.users table looks like 

    ```
    userdb=> select * from users;
    id | name | balance 
    ----+------+---------
      1 | foo  |    1000
      2 | bar  |    1500
      3 | bazz |    2000
    ```


4. Let's make some transactions for the users

    ```
    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/transactions/up -d '{ "id": "3", "amount": "100" }'

    // Output
    // {
    //  "balance": 2100
    // }

    // get the userbalance
    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/balance -d '{ "id": "3" }'

    // Output:
    // {
    //  "balance": 2100
    // }

    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/transactions/down -d '{ "id": "3", "amount": "500" }'

    / Output
    // {
    //  "balance": 1600
    // }

    // get the userbalance
    curl -X POST -H "CallType: GRPC_Call" http://www.simple-grpc-transcode.com/user/balance -d '{ "id": "3" }'

    // Output:
    // {
    //  "balance": 1600
    // }

    ```

    After those operations transactiondb.transaction_records table looks like 

    ```
    transactiondb=> select * from transaction_records ;
    user_id | amount | debited_or_credited | current_balance 
    ---------+--------+---------------------+-----------------
          3 |    100 | credited            |            2100
          3 |    500 | debited             |            1600
    ```

    and, userdb.users table looks like
    ```
    userdb=> select * from users;
    id | name | balance 
    ----+------+---------
      1 | foo  |    1000
      2 | bar  |    1500
      3 | bazz |    1600
    ```
**I have created a `UpdateUserBalance` `gRPC endpoint for` `user` service which is responsible for updating balance of a particular user. And during the transaction I have called this endpoint and retrive the `user_current_balance`.**