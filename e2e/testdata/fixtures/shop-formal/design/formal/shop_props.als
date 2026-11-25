-- Structural properties of the shop model. Hand-written.
open shop

-- Every order is owned by exactly one customer.
assert NoOrphanOrder { all o: Order | one c: Customer | o in c.order }

-- The customer an order names is the customer that lists it.
assert OwnerListsOrder { all o: Order | o in o.customer.order }

-- Reach predicates.
pred SomeOrder { some Order }
pred TwoBusyCustomers { #Customer >= 2 and all c: Customer | some c.order }
