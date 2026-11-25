-- Who may write which record, and the structural rules it rests on.
-- Hand-written; machinery never overwrites this file.
open crm

-- The write policy as relations over the model. A Deal is written by its
-- owner or an Admin, a Task by its assignee or an Admin, and only an Admin
-- reopens a lost Deal.
fun mayWriteDeal: User -> Deal { {u: User, d: Deal | d.owner = u or u.role = Admin} }
fun mayWriteTask: User -> Task { {u: User, t: Task | t.assignee = u or u.role = Admin} }
fun mayReopen: User -> Deal { {u: User, d: Deal | u.role = Admin and d.stage = Lost} }

-- Every Deal has exactly one owner.
assert DealHasOneOwner { all d: Deal | one d.owner }

-- Every Task belongs to exactly one Deal.
assert TaskInOneDeal { all t: Task | one t.deal }

-- Every Task has exactly one assignee.
assert TaskHasOneAssignee { all t: Task | one t.assignee }

-- Nobody who is neither the owner nor an Admin can write a Deal.
assert OnlyOwnerOrAdminWritesDeal {
  all u: User, d: Deal | (d.owner != u and u.role != Admin) implies u->d not in mayWriteDeal
}

-- Nobody who is neither the assignee nor an Admin can write a Task.
assert OnlyAssigneeOrAdminWritesTask {
  all u: User, t: Task | (t.assignee != u and u.role != Admin) implies u->t not in mayWriteTask
}

-- Only an Admin reopens a Deal.
assert OnlyAdminReopens { all u: User, d: Deal | u->d in mayReopen implies u.role = Admin }

-- Reach predicates.
pred RepOwnsDeal { some d: Deal | d.owner.role = Rep }
pred IdleAdminWrites {
  some u: User | u.role = Admin and no owner.u and no assignee.u and some u.mayWriteDeal and some u.mayWriteTask
}
