# Microservices gopher

This project is a Go-based microservices architecture for an e-commerce system, consisting of four independent services: User , Order, Payment, and Notification. The User Service handles user creation , while the Order Service manages order creation and updates. The Payment Service integrates with Stripe to process transactions, and the Notification Service sends email updates using SMTP.

## Design

<img width="1318" alt="Screenshot 2025-02-22 at 12 06 53 PM" src="https://github.com/user-attachments/assets/7547d795-f682-4e8e-9ccf-25b84ec8e87c" />



## Video



## Installation

### For Building Image
```bash
cd to root 
```
```bash
docker-compose build
```

## Run Locally

Clone the project

Go to the Root

Install dependencies

```bash
 docker-compose up --build
```

Install dependencies

```bash
  go mod tidy
```

would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)








