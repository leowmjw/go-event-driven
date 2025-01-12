# SCENARIO

## Objective

Demo how a mid-level complex multi-party system can be built using event driven system

## Services / Bounded Context

High level description of the Bounded Context for an e-commerce example

### Customers Management Context

- Simple CRUD to add Customers into the system
- Show how complexity of Outbox Pattern avoided is just do the CRUD in Temporal itself
- Sync/Dump from Dimension data

### Ordering Context

- CRUD for Order; with related Customer Info for Delivery, Billing Projected
- Projection of Order Status, Ordering History, Dispute Management
- Uses Orchestration for Critical Flow; of Payment + Fulfillment + Delivery + Refund

### Analyrics + ML Context

- Feature Engineering to simple prediction of fraud
- Monthly Report of Sales + Disputes

### Inventory Context

- CRUD for Inventory; with related Product Info Available for Purchase, which are being delivered in, which are being shipepd out
- Projection of Inventory Status and latest number to be used by Ordering Context
- Uses Orchestration for Critical Flow; of Order Fulfillment + Delivery + Inventory Management + Refund