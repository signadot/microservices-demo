<p align="center">
<img src="src/frontend/static/icons/Hipster_HeroLogoCyan.svg" width="300" alt="Online Boutique" />
</p>


**[Online Boutique](https://microservices.honeydemo.io)** is a cloud-native
microservices demo application. Online Boutique consists of a 10-tier
microservices application, written in 5 different languages: Go, Java, .NET,
Node, and Python. The application is a web-based e-commerce platform where users
can browse items, add them to a cart, and purchase them.

**[Signadot](https://signadot.com)** uses this application to demonstrate the
use of technologies like Kubernetes, gRPC, and OpenTelemetry. This application
works on any Kubernetes cluster. It’s **easy to deploy with little to no
configuration**.

## Screenshots

| Home Page                                                                                                               | Checkout Screen                                                                                                          |
|-------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| [![Screenshot of store homepage](./docs/img/online-boutique-frontend-1.png)](./docs/img/online-boutique-frontend-1.png) | [![Screenshot of checkout screen](./docs/img/online-boutique-frontend-2.png)](./docs/img/online-boutique-frontend-2.png) |

## OpenTelemetry

Online Boutique is instrumented using the
[OpenTelemetry](https://opentelemetry.io) framework. There are simple and
advanced instrumentation techniques offered by OpenTelemetry that are leveraged
in the application. Each service in the [src](./src) folder explains how
OpenTelemetry was used with specific code examples.

## Development

### Prerequisites

- [Docker for Desktop](https://www.docker.com/products/docker-desktop)
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl), a CLI to interact
  with Kubernetes
- [skaffold]( https://skaffold.dev/docs/install/), a tool that builds and
  deploys Docker images in bulk
- [Helm](https://helm.sh), a package manager for Kubernetes

### Install

1. Launch a local Kubernetes cluster with one of the following options:

- To launch **Minikube** (tested with Ubuntu Linux). Please, ensure that the
  local Kubernetes cluster has at least:
    - 4 CPUs
    - 4.0 GiB memory
    - 32 GB disk space
```shell
minikube start --cpus=4 --memory 4096 --disk-size 32g
```

- To launch **Docker for Desktop** (tested with Mac/Windows). Go to Preferences:
    - choose “Enable Kubernetes”,
    - set CPUs to at least 3, and Memory to at least 6.0 GiB
    - on the "Disk" tab, set at least 32 GB disk space

2. Run `kubectl get nodes` to verify you're connected to the respective control
   plane.

3. Run `skaffold run` (first time will be slow, it can take ~20 minutes). This
   will build and deploy the application. If you need to rebuild the images
   automatically as you refactor the code, run `skaffold dev` command.

4. Run `kubectl get pods` to verify the Pods are ready and running.

5. Access the web frontend through your browser

- **Minikube** requires you to run a command to access the frontend service:
```shell
minikube service frontend-external
```

- **Docker For Desktop** should automatically provide the frontend at
  http://localhost:80


### Cleanup

If you've deployed the application with `skaffold run` command, you can run
`skaffold delete` to clean up the deployed resources.

## Architecture

**Online Boutique** is composed of 10 microservices (plus a load generator)
written in 5 different languages that communicate with each other over gRPC.

[![Architecture of
microservices](./docs/img/architecture-diagram.png)](./docs/img/architecture-diagram.png)

Find **Protocol Buffers Descriptions** at the [`./pb` directory](./pb).

| Service                                              | Language      | Description                                                                                                                       |
|------------------------------------------------------|---------------|-----------------------------------------------------------------------------------------------------------------------------------|
| [adservice](./src/adservice)                         | Java          | Provides text ads based on given context words.                                                                                   |
| [cartservice](./src/cartservice)                     | C#            | Stores the items in the user's shopping cart in Redis and retrieves it.                                                           |
| [checkoutservice](./src/checkoutservice)             | Go            | Retrieves user cart, prepares order and orchestrates the payment, shipping and the email notification.                            |
| [currencyservice](./src/currencyservice)             | Node.js       | Converts one money amount to another currency. Uses real values fetched from European Central Bank. It's the highest QPS service. |
| [emailservice](./src/emailservice)                   | Python        | Sends users an order confirmation email (mock).                                                                                   |
| [frontend](./src/frontend)                           | Go            | Exposes an HTTP server to serve the website. Does not require signup/login and generates session IDs for all users automatically. |
| [loadgenerator](./src/loadgenerator)                 | Python/Locust | Continuously sends requests imitating realistic user shopping flows to the frontend.                                              |
| [paymentservice](./src/paymentservice)               | Node.js       | Charges the given credit card info (mock) with the given amount and returns a transaction ID.                                     |
| [productcatalogservice](./src/productcatalogservice) | Go            | Provides the list of products from a JSON file and ability to search products and get individual products.                        |
| [recommendationservice](./src/recommendationservice) | Python        | Recommends other products based on what's given in the cart.                                                                      |
| [shippingservice](./src/shippingservice)             | Go            | Gives shipping cost estimates based on the shopping cart. Ships items to the given address (mock)                                 |

## Features

- **[Kubernetes](https://kubernetes.io):** The app is designed to run on
  Kubernetes
- **[gRPC](https://grpc.io):** Microservices use a high volume of gRPC calls to
  communicate to each other.
- **[OpenTelemetry](https://opentelemetry.io/) Tracing:** Most services are
  instrumented using OpenTelemetry trace providers for gRPC/HTTP.
- **[Skaffold](https://skaffold.dev):** Application is deployed to Kubernetes
  with a single command using Skaffold.
- **Synthetic Load Generation:** The application demo comes with a background
  job that creates realistic usage patterns on the website using
  [Locust](https://locust.io/) load generator.

## History

This project originated from Honeycomb's [Microservices
Demo](https://github.com/honeycombio/microservices-demo/), which itself was
derived from the excellent Google Cloud Platform [Microservices
Demo](https://github.com/GoogleCloudPlatform/microservices-demo). It was forked
in 2022.

## Application demo

This application will exhibit usage of Sandbox environments using the
[Signadot](https://signadot.com) platform. See [feature testing
guide](https://github.com/signadot/microservices-demo/blob/main/docs/feature-testing.md)
for more details.


---

This is not an official Signadot or Google project.
