# Checkout Backend

A simple checkout system backend API which supports listing, adding and purchasing products.

## Usage

Available endpoints:

- `GET /product` - Retrieves a list of products
- `POST /product` - Add a new product
- `POST /purchase` - Purchase available items

## Improvements

- Use ORM libraries instead of the database/sql package. This allows more abstraction/simplification of database interactions and can offer improved security.
- Utilize database migration to version control database changes.
- Utilize workers and queues (message driven architecture). Allows processes to be de-coupled as well as to be scaled up independently of other processes. Message driven architecture can also improve reliability as messages can be re-tried if a worker is not able to process a message currently.
- Add logging and tracing to make it easier to debug problems.
- Add more unit tests
