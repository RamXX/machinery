//go:build ladybug

// This file is the ONLY importer of go-ladybug (BUILD.md 4.5, 9; verified by
// C-ARCH-01). Build against an installed Ladybug C library with the additional
// system_ladybug tag.
package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	lbug "github.com/LadybugDB/go-ladybug"

	"crm/internal/model"
)

const queryTimeoutMillis = 10_000

type ladybugRepo struct {
	mu          sync.Mutex
	defaultPath string
	fault       func(operation string) error
}

type ladybugTx struct {
	db      *lbug.Database
	conn    *lbug.Connection
	path    string
	mu      sync.Mutex
	started bool
	ended   bool
	closed  bool
}

var writeOwners = struct {
	sync.Mutex
	paths map[string]*ladybugTx
}{paths: make(map[string]*ladybugTx)}

func NewLadybug() Repo { return &ladybugRepo{} }

// NewLadybugWithFault constructs the real repository with a deterministic
// pre-operation fault seam. It exists for the three boundary cases that the
// operating system cannot reproduce portably (conflict, disk-full, timeout).
func NewLadybugWithFault(fault func(operation string) error) Repo {
	return &ladybugRepo{fault: fault}
}

func (r *ladybugRepo) before(operation string) error {
	if r.fault == nil {
		return nil
	}
	return mapErr(r.fault(operation))
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	var kind error
	switch {
	case strings.Contains(msg, "interrupt"), strings.Contains(msg, "timeout"):
		kind = model.ErrTimeout
	case strings.Contains(msg, "no space"), strings.Contains(msg, "disk full"), strings.Contains(msg, "maximum database size"):
		kind = model.ErrDiskFull
	case strings.Contains(msg, "conflict"):
		kind = model.ErrConflict
	case strings.Contains(msg, "lock"), strings.Contains(msg, "another process"):
		kind = model.ErrLocked
	case strings.Contains(msg, "constraint"), strings.Contains(msg, "duplicate"), strings.Contains(msg, "primary key"):
		kind = model.ErrConstraint
	case strings.Contains(msg, "corrupt"), strings.Contains(msg, "catalog"), strings.Contains(msg, "version"):
		kind = model.ErrCorrupt
	default:
		kind = model.ErrUnavailable
	}
	return fmt.Errorf("%w: %v", kind, err)
}

var schema = []string{
	"CREATE NODE TABLE IF NOT EXISTS CRMUser(id STRING, username STRING, passwordHash STRING, role STRING, status STRING, createdAt STRING, teamId STRING, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Team(id STRING, name STRING, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Account(id STRING, name STRING, domain STRING, industry STRING, ownerId STRING, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Contact(id STRING, fullName STRING, email STRING, phone STRING, title STRING, ownerId STRING, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Deal(id STRING, title STRING, amountCents INT64, stage STRING, closeDate STRING, ownerId STRING, contactIds STRING, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Pipeline(id STRING, name STRING, isDefault BOOL, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Activity(id STRING, type STRING, subject STRING, body STRING, occurredAt STRING, ownerId STRING, contactId STRING, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Task(id STRING, title STRING, dueDate STRING, status STRING, ownerId STRING, dealId STRING, PRIMARY KEY(id))",
	"CREATE NODE TABLE IF NOT EXISTS Tag(id STRING, name STRING, color STRING, PRIMARY KEY(id))",
}

func (r *ladybugRepo) Open(path string) (Tx, error) {
	r.mu.Lock()
	if path == "" {
		path = r.defaultPath
	} else {
		r.defaultPath = path
	}
	r.mu.Unlock()
	if path == "" {
		return nil, fmt.Errorf("%w: no database path configured", model.ErrUnavailable)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve database path: %v", model.ErrUnavailable, err)
	}
	writeOwners.Lock()
	owner := writeOwners.paths[abs]
	writeOwners.Unlock()
	if owner != nil {
		return nil, model.ErrLocked
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return nil, fmt.Errorf("%w: create database parent directory: %v", model.ErrUnavailable, err)
	}
	if info, err := os.Lstat(abs); err == nil && info.IsDir() {
		if catalog, catalogErr := os.Lstat(filepath.Join(abs, "catalog.kz")); catalogErr == nil && catalog.Size() < 32 {
			return nil, fmt.Errorf("%w: invalid catalog", model.ErrCorrupt)
		} else if catalogErr != nil && !errors.Is(catalogErr, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: inspect database catalog: %v", model.ErrUnavailable, catalogErr)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: inspect database path: %v", model.ErrUnavailable, err)
	}
	db, err := lbug.OpenDatabase(abs, lbug.DefaultSystemConfig())
	if err != nil {
		return nil, mapErr(err)
	}
	conn, err := lbug.OpenConnection(db)
	if err != nil {
		db.Close()
		return nil, mapErr(err)
	}
	conn.SetTimeout(queryTimeoutMillis)
	tx := &ladybugTx{db: db, conn: conn, path: abs}
	for _, statement := range schema {
		if err := tx.exec(statement, nil); err != nil {
			conn.Close()
			db.Close()
			return nil, err
		}
	}
	return tx, nil
}

func asLadybugTx(tx Tx) (*ladybugTx, error) {
	t, ok := tx.(*ladybugTx)
	if !ok || t == nil || t.conn == nil || t.closed {
		return nil, fmt.Errorf("%w: invalid transaction handle", model.ErrUnavailable)
	}
	return t, nil
}

func (r *ladybugRepo) BeginWrite(raw Tx) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started && !t.ended {
		return model.ErrConflict
	}
	writeOwners.Lock()
	if owner := writeOwners.paths[t.path]; owner != nil && owner != t {
		writeOwners.Unlock()
		return model.ErrLocked
	}
	writeOwners.paths[t.path] = t
	writeOwners.Unlock()
	if err := t.exec("BEGIN TRANSACTION", nil); err != nil {
		releaseWriteOwner(t)
		return err
	}
	t.started, t.ended = true, false
	return nil
}

func releaseWriteOwner(t *ladybugTx) {
	writeOwners.Lock()
	if writeOwners.paths[t.path] == t {
		delete(writeOwners.paths, t.path)
	}
	writeOwners.Unlock()
}

func (r *ladybugRepo) Commit(raw Tx) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	if t.started && !t.ended {
		err = t.exec("COMMIT", nil)
	}
	t.ended = true
	releaseWriteOwner(t)
	t.close()
	return err
}

func (r *ladybugRepo) Rollback(raw Tx) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	if t.started && !t.ended {
		err = t.exec("ROLLBACK", nil)
	}
	t.ended = true
	releaseWriteOwner(t)
	t.close()
	return err
}

func (t *ladybugTx) close() {
	if t.closed {
		return
	}
	t.conn.Close()
	t.db.Close()
	t.closed = true
}

func (t *ladybugTx) exec(query string, args map[string]any) error {
	stmt, err := t.conn.Prepare(query)
	if err != nil {
		return mapErr(err)
	}
	defer stmt.Close()
	result, err := t.conn.Execute(stmt, args)
	if err != nil {
		return mapErr(err)
	}
	result.Close()
	return nil
}

func (t *ladybugTx) one(query string, args map[string]any) ([]any, error) {
	stmt, err := t.conn.Prepare(query)
	if err != nil {
		return nil, mapErr(err)
	}
	defer stmt.Close()
	result, err := t.conn.Execute(stmt, args)
	if err != nil {
		return nil, mapErr(err)
	}
	defer result.Close()
	if !result.HasNext() {
		return nil, model.ErrNotFound
	}
	tuple, err := result.Next()
	if err != nil {
		return nil, mapErr(err)
	}
	defer tuple.Close()
	values, err := tuple.GetAsSlice()
	if err != nil {
		return nil, mapErr(err)
	}
	return values, nil
}

func timeString(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339Nano)
}

func pointerTimeString(v *time.Time) string {
	if v == nil {
		return ""
	}
	return timeString(*v)
}

func parseTime(value any) time.Time {
	v, _ := time.Parse(time.RFC3339Nano, fmt.Sprint(value))
	return v
}

func parsePointerTime(value any) *time.Time {
	if fmt.Sprint(value) == "" {
		return nil
	}
	v := parseTime(value)
	return &v
}

func (r *ladybugRepo) GetUserByName(raw Tx, name string) (model.User, error) {
	t, err := asLadybugTx(raw)
	if err != nil {
		return model.User{}, err
	}
	return getUser(t, "MATCH (u:CRMUser) WHERE u.username = $value RETURN u.id, u.username, u.passwordHash, u.role, u.status, u.createdAt, u.teamId LIMIT 1", name)
}

func (r *ladybugRepo) GetUser(raw Tx, id string) (model.User, error) {
	t, err := asLadybugTx(raw)
	if err != nil {
		return model.User{}, err
	}
	return getUser(t, "MATCH (u:CRMUser) WHERE u.id = $value RETURN u.id, u.username, u.passwordHash, u.role, u.status, u.createdAt, u.teamId LIMIT 1", id)
}

func getUser(t *ladybugTx, query, value string) (model.User, error) {
	v, err := t.one(query, map[string]any{"value": value})
	if err != nil {
		return model.User{}, err
	}
	return model.User{ID: fmt.Sprint(v[0]), Username: fmt.Sprint(v[1]), PasswordHash: fmt.Sprint(v[2]), Role: model.UserRole(fmt.Sprint(v[3])), Status: model.UserStatus(fmt.Sprint(v[4])), CreatedAt: parseTime(v[5]), TeamID: fmt.Sprint(v[6])}, nil
}

func (r *ladybugRepo) GetDeal(raw Tx, id string) (model.Deal, error) {
	t, err := asLadybugTx(raw)
	if err != nil {
		return model.Deal{}, err
	}
	v, err := t.one("MATCH (d:Deal) WHERE d.id = $id RETURN d.id, d.title, d.amountCents, d.stage, d.closeDate, d.ownerId, d.contactIds LIMIT 1", map[string]any{"id": id})
	if err != nil {
		return model.Deal{}, err
	}
	var contacts []string
	_ = json.Unmarshal([]byte(fmt.Sprint(v[6])), &contacts)
	amount, _ := v[2].(int64)
	return model.Deal{ID: fmt.Sprint(v[0]), Title: fmt.Sprint(v[1]), AmountCents: amount, Stage: model.DealStage(fmt.Sprint(v[3])), CloseDate: parsePointerTime(v[4]), OwnerID: fmt.Sprint(v[5]), ContactIDs: contacts}, nil
}

func (r *ladybugRepo) GetTask(raw Tx, id string) (model.Task, error) {
	t, err := asLadybugTx(raw)
	if err != nil {
		return model.Task{}, err
	}
	v, err := t.one("MATCH (n:Task) WHERE n.id = $id RETURN n.id, n.title, n.dueDate, n.status, n.ownerId, n.dealId LIMIT 1", map[string]any{"id": id})
	if err != nil {
		return model.Task{}, err
	}
	return model.Task{ID: fmt.Sprint(v[0]), Title: fmt.Sprint(v[1]), DueDate: parsePointerTime(v[2]), Status: model.TaskStatus(fmt.Sprint(v[3])), OwnerID: fmt.Sprint(v[4]), DealID: fmt.Sprint(v[5])}, nil
}

func (r *ladybugRepo) GetAccount(raw Tx, id string) (model.Account, error) {
	t, err := asLadybugTx(raw)
	if err != nil {
		return model.Account{}, err
	}
	v, err := t.one("MATCH (n:Account) WHERE n.id = $id RETURN n.id, n.name, n.domain, n.industry, n.ownerId LIMIT 1", map[string]any{"id": id})
	if err != nil {
		return model.Account{}, err
	}
	return model.Account{ID: fmt.Sprint(v[0]), Name: fmt.Sprint(v[1]), Domain: fmt.Sprint(v[2]), Industry: fmt.Sprint(v[3]), OwnerID: fmt.Sprint(v[4])}, nil
}

func (r *ladybugRepo) GetContact(raw Tx, id string) (model.Contact, error) {
	t, err := asLadybugTx(raw)
	if err != nil {
		return model.Contact{}, err
	}
	v, err := t.one("MATCH (n:Contact) WHERE n.id = $id RETURN n.id, n.fullName, n.email, n.phone, n.title, n.ownerId LIMIT 1", map[string]any{"id": id})
	if err != nil {
		return model.Contact{}, err
	}
	return model.Contact{ID: fmt.Sprint(v[0]), FullName: fmt.Sprint(v[1]), Email: fmt.Sprint(v[2]), Phone: fmt.Sprint(v[3]), Title: fmt.Sprint(v[4]), OwnerID: fmt.Sprint(v[5])}, nil
}

func exists(t *ladybugTx, label, id string) (bool, error) {
	_, err := t.one("MATCH (n:"+label+") WHERE n.id = $id RETURN n.id LIMIT 1", map[string]any{"id": id})
	if errors.Is(err, model.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func save(t *ladybugTx, label, id, create, update string, args map[string]any, collisionIsConstraint bool) error {
	if id == "" {
		return fmt.Errorf("%w: empty id", model.ErrConstraint)
	}
	present, err := exists(t, label, id)
	if err != nil {
		return err
	}
	if present && collisionIsConstraint {
		return model.ErrConstraint
	}
	if present {
		return t.exec(update, args)
	}
	return t.exec(create, args)
}

func (r *ladybugRepo) SaveDeal(raw Tx, d model.Deal) error {
	if err := r.before("SaveDeal"); err != nil {
		return err
	}
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if d.OwnerID == "" || d.AmountCents < 0 || (d.Stage == model.StageWon && d.CloseDate == nil) {
		return model.ErrConstraint
	}
	contacts, _ := json.Marshal(d.ContactIDs)
	args := map[string]any{"id": d.ID, "title": d.Title, "amount": d.AmountCents, "stage": string(d.Stage), "close": pointerTimeString(d.CloseDate), "owner": d.OwnerID, "contacts": string(contacts)}
	present, err := exists(t, "Deal", d.ID)
	if err != nil {
		return err
	}
	if present {
		old, getErr := r.GetDeal(raw, d.ID)
		if getErr != nil {
			return getErr
		}
		if old.ID == d.ID && old.Title == d.Title && old.AmountCents == d.AmountCents && old.Stage == d.Stage && old.OwnerID == d.OwnerID && pointerTimeString(old.CloseDate) == pointerTimeString(d.CloseDate) {
			return model.ErrConstraint
		}
		return t.exec("MATCH (n:Deal) WHERE n.id=$id SET n.title=$title, n.amountCents=$amount, n.stage=$stage, n.closeDate=$close, n.ownerId=$owner, n.contactIds=$contacts", args)
	}
	return t.exec("CREATE (:Deal {id:$id, title:$title, amountCents:$amount, stage:$stage, closeDate:$close, ownerId:$owner, contactIds:$contacts})", args)
}

func (r *ladybugRepo) SaveTask(raw Tx, v model.Task) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if v.OwnerID == "" {
		return model.ErrConstraint
	}
	a := map[string]any{"id": v.ID, "title": v.Title, "due": pointerTimeString(v.DueDate), "status": string(v.Status), "owner": v.OwnerID, "deal": v.DealID}
	return save(t, "Task", v.ID, "CREATE (:Task {id:$id,title:$title,dueDate:$due,status:$status,ownerId:$owner,dealId:$deal})", "MATCH (n:Task) WHERE n.id=$id SET n.title=$title,n.dueDate=$due,n.status=$status,n.ownerId=$owner,n.dealId=$deal", a, false)
}

func (r *ladybugRepo) SaveUser(raw Tx, v model.User) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if v.Username == "" || (v.Role == model.RoleManager && v.TeamID == "") {
		return model.ErrConstraint
	}
	if row, qerr := t.one("MATCH (n:CRMUser) WHERE n.username=$name AND n.id<>$id RETURN n.id LIMIT 1", map[string]any{"name": v.Username, "id": v.ID}); qerr == nil && len(row) > 0 {
		return model.ErrConstraint
	} else if qerr != nil && !errors.Is(qerr, model.ErrNotFound) {
		return qerr
	}
	a := map[string]any{"id": v.ID, "name": v.Username, "hash": v.PasswordHash, "role": string(v.Role), "status": string(v.Status), "created": timeString(v.CreatedAt), "team": v.TeamID}
	return save(t, "CRMUser", v.ID, "CREATE (:CRMUser {id:$id,username:$name,passwordHash:$hash,role:$role,status:$status,createdAt:$created,teamId:$team})", "MATCH (n:CRMUser) WHERE n.id=$id SET n.username=$name,n.passwordHash=$hash,n.role=$role,n.status=$status,n.createdAt=$created,n.teamId=$team", a, false)
}

func (r *ladybugRepo) SaveAccount(raw Tx, v model.Account) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if v.OwnerID == "" {
		return model.ErrConstraint
	}
	a := map[string]any{"id": v.ID, "name": v.Name, "domain": v.Domain, "industry": v.Industry, "owner": v.OwnerID}
	return save(t, "Account", v.ID, "CREATE (:Account {id:$id,name:$name,domain:$domain,industry:$industry,ownerId:$owner})", "MATCH (n:Account) WHERE n.id=$id SET n.name=$name,n.domain=$domain,n.industry=$industry,n.ownerId=$owner", a, false)
}

func (r *ladybugRepo) SaveContact(raw Tx, v model.Contact) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if v.OwnerID == "" {
		return model.ErrConstraint
	}
	a := map[string]any{"id": v.ID, "name": v.FullName, "email": v.Email, "phone": v.Phone, "title": v.Title, "owner": v.OwnerID}
	return save(t, "Contact", v.ID, "CREATE (:Contact {id:$id,fullName:$name,email:$email,phone:$phone,title:$title,ownerId:$owner})", "MATCH (n:Contact) WHERE n.id=$id SET n.fullName=$name,n.email=$email,n.phone=$phone,n.title=$title,n.ownerId=$owner", a, false)
}

func (r *ladybugRepo) SaveActivity(raw Tx, v model.Activity) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if v.OwnerID == "" {
		return model.ErrConstraint
	}
	a := map[string]any{"id": v.ID, "type": string(v.Type), "subject": v.Subject, "body": v.Body, "occurred": timeString(v.OccurredAt), "owner": v.OwnerID, "contact": v.ContactID}
	return save(t, "Activity", v.ID, "CREATE (:Activity {id:$id,type:$type,subject:$subject,body:$body,occurredAt:$occurred,ownerId:$owner,contactId:$contact})", "", a, true)
}

func (r *ladybugRepo) SavePipeline(raw Tx, v model.Pipeline) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	a := map[string]any{"id": v.ID, "name": v.Name, "isDefaultValue": v.IsDefault}
	return save(t, "Pipeline", v.ID, "CREATE (:Pipeline {id:$id,name:$name,isDefault:$isDefaultValue})", "MATCH (n:Pipeline) WHERE n.id=$id SET n.name=$name,n.isDefault=$isDefaultValue", a, false)
}

func (r *ladybugRepo) SaveTag(raw Tx, v model.Tag) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if row, qerr := t.one("MATCH (n:Tag) WHERE n.name=$name AND n.id<>$id RETURN n.id LIMIT 1", map[string]any{"name": v.Name, "id": v.ID}); qerr == nil && len(row) > 0 {
		return model.ErrConstraint
	} else if qerr != nil && !errors.Is(qerr, model.ErrNotFound) {
		return qerr
	}
	a := map[string]any{"id": v.ID, "name": v.Name, "color": v.Color}
	return save(t, "Tag", v.ID, "CREATE (:Tag {id:$id,name:$name,color:$color})", "MATCH (n:Tag) WHERE n.id=$id SET n.name=$name,n.color=$color", a, false)
}

func (r *ladybugRepo) SaveTeam(raw Tx, v model.Team) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if row, qerr := t.one("MATCH (n:Team) WHERE n.name=$name AND n.id<>$id RETURN n.id LIMIT 1", map[string]any{"name": v.Name, "id": v.ID}); qerr == nil && len(row) > 0 {
		return model.ErrConstraint
	} else if qerr != nil && !errors.Is(qerr, model.ErrNotFound) {
		return qerr
	}
	a := map[string]any{"id": v.ID, "name": v.Name}
	return save(t, "Team", v.ID, "CREATE (:Team {id:$id,name:$name})", "MATCH (n:Team) WHERE n.id=$id SET n.name=$name", a, false)
}

func (r *ladybugRepo) SetDefaultPipeline(raw Tx, id string) error {
	t, err := asLadybugTx(raw)
	if err != nil {
		return err
	}
	if ok, err := exists(t, "Pipeline", id); err != nil {
		return err
	} else if !ok {
		return model.ErrNotFound
	}
	if err := t.exec("MATCH (n:Pipeline) SET n.isDefault=false", nil); err != nil {
		return err
	}
	return t.exec("MATCH (n:Pipeline) WHERE n.id=$id SET n.isDefault=true", map[string]any{"id": id})
}

func (r *ladybugRepo) CountDefaultPipelines(raw Tx) (int, error) {
	t, err := asLadybugTx(raw)
	if err != nil {
		return 0, err
	}
	v, err := t.one("MATCH (n:Pipeline) WHERE n.isDefault=true RETURN count(n)", nil)
	if err != nil {
		return 0, err
	}
	switch n := v[0].(type) {
	case int64:
		return int(n), nil
	case uint64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("%w: invalid count type %T", model.ErrCorrupt, v[0])
	}
}
