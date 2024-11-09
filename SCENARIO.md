# SCENARIO

## Objective

Demo how a mid-level complex multi-party system can be built using event driven system

## Services / Bounded Context

High level description of the Bounded Context for an e-commerce example

### Customers Management Context

- Simple CRUD to add Customers into the system

### Ordering Context

- CRUD for Order; with related Customer Info for Delivery, Billing Projected
- Projection of Order Status, Ordering History, Dispute Management
- Uses Orchestration for Critical Flow; of Payment + Fulfillment + Delivery + Refund

### Analyrics + ML Context

- Feature Engineering to simple prediction of fraud
- Monthly Report of Sales + Disputes
