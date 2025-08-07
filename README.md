# go-eventdriven-arch

sequenceDiagram
    participant Client
    participant OrderService
    participant OrderCreatedHandler
    participant InventoryService
    participant NotificationHandler
    participant OrderHandler

    Client->>OrderService: Create Order
    OrderService->>OrderService: Save Order (status: Pending)
    OrderService->>+OrderCreatedHandler: Publish OrderCreated Event
    
    OrderCreatedHandler->>InventoryService: Reserve Product
    alt Product Available
        InventoryService-->>OrderCreatedHandler: Success
        OrderCreatedHandler->>OrderCreatedHandler: Update Order (status: Confirmed)
        OrderCreatedHandler->>+NotificationHandler: Publish InventoryStatusUpdated(HasStock=true, OrderID)
    else Product Unavailable
        InventoryService-->>OrderCreatedHandler: Failed
        OrderCreatedHandler->>+NotificationHandler: Publish InventoryStatusUpdated(HasStock=false, OrderID)
    end
    
    NotificationHandler->>NotificationHandler: Send Notification
    NotificationHandler->>+OrderHandler: Publish NotificationSent(OrderID)
    OrderHandler->>OrderHandler: Update Order (notificationStatus: sent)
