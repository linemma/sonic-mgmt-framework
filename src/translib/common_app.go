package translib

import (
	"errors"
	"fmt"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
	"reflect"
	"translib/db"
	"translib/tlerr"
	"translib/transformer"
)

var ()

type CommonApp struct {
	pathInfo       *PathInfo
	ygotRoot       *ygot.GoStruct
	ygotTarget     *interface{}
	cmnAppTableMap map[string]map[string]db.Value
}

var cmnAppInfo = appInfo{appType: reflect.TypeOf(CommonApp{}),
	ygotRootType:  nil,
	isNative:      false,
	tablesToWatch: nil}

func init() {

	// @todo : Optimize to register supported paths/yang via common app and report error for unsupported
	register_model_path := []string{"/sonic-"} // register yang model path(s) to be supported via common app
	for _, mdl_pth := range register_model_path {
		err := register(mdl_pth, &cmnAppInfo)

		if err != nil {
			log.Fatal("Register Common app module with App Interface failed with error=", err, "for path=", mdl_pth)
		}
	}

}

func (app *CommonApp) initialize(data appData) {
	log.Info("initialize:path =", data.path)
	pathInfo := NewPathInfo(data.path)
	*app = CommonApp{pathInfo: pathInfo, ygotRoot: data.ygotRoot, ygotTarget: data.ygotTarget}

}

func (app *CommonApp) translateCreate(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	log.Info("translateCreate:path =", app.pathInfo.Path)

	keys, err = app.translateCRUCommon(d, CREATE)

	return keys, err
}

func (app *CommonApp) translateUpdate(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	log.Info("translateUpdate:path =", app.pathInfo.Path)

	keys, err = app.translateCRUCommon(d, UPDATE)

	return keys, err
}

func (app *CommonApp) translateReplace(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	log.Info("translateReplace:path =", app.pathInfo.Path)

	keys, err = app.translateCRUCommon(d, REPLACE)

	return keys, err
}

func (app *CommonApp) translateDelete(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	log.Info("translateDelete:path =", app.pathInfo.Path)
	keys, err = app.translateCRUCommon(d, DELETE)

	return keys, err
}

func (app *CommonApp) translateGet(dbs [db.MaxDB]*db.DB) error {
	var err error
	log.Info("translateGet:path =", app.pathInfo.Path)
	return err
}

func (app *CommonApp) translateSubscribe(dbs [db.MaxDB]*db.DB, path string) (*notificationOpts, *notificationInfo, error) {
	err := errors.New("Not supported")
	notifInfo := notificationInfo{dbno: db.ConfigDB}
	return nil, &notifInfo, err
}

func (app *CommonApp) processCreate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	log.Info("processCreate:path =", app.pathInfo.Path)
	targetType := reflect.TypeOf(*app.ygotTarget)
	log.Infof("processCreate: Target object is a <%s> of Type: %s", targetType.Kind().String(), targetType.Elem().Name())
	if err = app.processCommon(d, CREATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}

	return resp, err
}

func (app *CommonApp) processUpdate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse
	log.Info("processUpdate:path =", app.pathInfo.Path)
	if err = app.processCommon(d, UPDATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}

	return resp, err
}

func (app *CommonApp) processReplace(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse
	log.Info("processReplace:path =", app.pathInfo.Path)
	if err = app.processCommon(d, REPLACE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *CommonApp) processDelete(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	log.Info("processDelete:path =", app.pathInfo.Path)

	if err = app.processCommon(d, DELETE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}

	return resp, err
}

func (app *CommonApp) processGet(dbs [db.MaxDB]*db.DB) (GetResponse, error) {
	var err error
	var payload []byte
	log.Info("processGet:path =", app.pathInfo.Path)

	keySpec, err := transformer.XlateUriToKeySpec(app.pathInfo.Path, app.ygotRoot, app.ygotTarget)

	// table.key.fields
	var result = make(map[string]map[string]db.Value)

	for dbnum, specs := range *keySpec {
		for _, spec := range specs {
			err := transformer.TraverseDb(dbs[dbnum], spec, &result, nil)
			if err != nil {
				return GetResponse{Payload: payload}, err
			}
		}
	}

	payload, err = transformer.XlateFromDb(app.pathInfo.Path, result)
	if err != nil {
		return GetResponse{Payload: payload, ErrSrc: AppErr}, err
	}

	return GetResponse{Payload: payload}, err
}

func (app *CommonApp) translateCRUCommon(d *db.DB, opcode int) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	var tblsToWatch []*db.TableSpec
	log.Info("translateCRUCommon:path =", app.pathInfo.Path)

	// translate yang to db
	result, err := transformer.XlateToDb(app.pathInfo.Path, opcode, d, (*app).ygotRoot, (*app).ygotTarget)
	fmt.Println(result)
	log.Info("transformer.XlateToDb() returned", result)

	if err != nil {
		log.Error(err)
		return keys, err
	}
	if len(result) == 0 {
		log.Error("XlatetoDB() returned empty map")
		err = errors.New("transformer.XlatetoDB() returned empty map")
		return keys, err
	}
	app.cmnAppTableMap = result
	for tblnm, _ := range app.cmnAppTableMap {
		log.Error("Table name ", tblnm)
		tblsToWatch = append(tblsToWatch, &db.TableSpec{Name: tblnm})
	}
	log.Info("Tables to watch", tblsToWatch)

	cmnAppInfo.tablesToWatch = tblsToWatch

	keys, err = app.generateDbWatchKeys(d, false)

	return keys, err
}

func (app *CommonApp) processCommon(d *db.DB, opcode int) error {
	var err error
	err = app.cmnAppDataDbOperation(d, opcode, app.cmnAppTableMap)
	return err
}

func (app *CommonApp) cmnAppDataDbOperation(d *db.DB, opcode int, cmnAppDataDbMap map[string]map[string]db.Value) error {
	var err error
	log.Info("Processing DB operation for ", cmnAppDataDbMap)
	var cmnAppTs *db.TableSpec
	for tblNm, tblInst := range cmnAppDataDbMap {
		log.Info("Table name ", tblNm)
		cmnAppTs = &db.TableSpec{Name: tblNm}
		for tblKey, tblRw := range tblInst {
			log.Info("Table key and row ", tblKey, tblRw)
			switch opcode {
			case CREATE:
				log.Info("CREATE case")
				existingEntry, err := d.GetEntry(cmnAppTs, db.Key{Comp: []string{tblKey}})
				if existingEntry.IsPopulated() {
					log.Info("Entry already exists hence return error.")
					return tlerr.AlreadyExists("Entry %s already exists", tblKey)
				} else {
					err = d.CreateEntry(cmnAppTs, db.Key{Comp: []string{tblKey}}, tblRw)
					if err != nil {
						log.Error("CREATE case - d.CreateEntry() failure")
						return err
					}
				}
			case UPDATE:
				log.Info("UPDATE case")
				existingEntry, err := d.GetEntry(cmnAppTs, db.Key{Comp: []string{tblKey}})
				if existingEntry.IsPopulated() {
					log.Info("Entry already exists hence modifying it.")
					err = d.ModEntry(cmnAppTs, db.Key{Comp: []string{tblKey}}, tblRw)
					if err != nil {
						log.Error("UPDATE case - d.ModEntry() failure")
						return err
					}
				} else {
					log.Info("Entry to be modified does not exist hence return error.")
					return tlerr.NotFound("Entry %s to be modified does not exist.", tblKey)

				}

			case REPLACE:
				log.Info("REPLACE case")
				existingEntry, err := d.GetEntry(cmnAppTs, db.Key{Comp: []string{tblKey}})
				if existingEntry.IsPopulated() {
					log.Info("Entry already exists hence execute db.SetEntry")
					err := d.SetEntry(cmnAppTs, db.Key{Comp: []string{tblKey}}, tblRw)
					if err != nil {
						log.Error("REPLACE case - d.SetEntry() failure")
						return err
					}
				} else {
					log.Info("Entry doesn't exist hence create it.")
					err = d.CreateEntry(cmnAppTs, db.Key{Comp: []string{tblKey}}, tblRw)
					if err != nil {
						log.Error("REPLACE case - d.CreateEntry() failure")
						return err
					}
				}
				//TODO : should the table level replace be handled??
			case DELETE:
				log.Info("DELETE case")
				if len(tblRw.Field) == 0 {
					log.Info("DELETE case - no fields/cols to delete hence delete the entire row.")
					err := d.DeleteEntry(cmnAppTs, db.Key{Comp: []string{tblKey}})
					if err != nil {
						log.Error("DELETE case - d.DeleteEntry() failure")
						return err
					}
				} else {
					log.Info("DELETE case - fields/cols to delete hence delete only those fields.")
					err := d.DeleteEntryFields(cmnAppTs, db.Key{Comp: []string{tblKey}}, tblRw)
					if err != nil {
						log.Error("DELETE case - d.DeleteEntryFields() failure")
						return err
					}
				}
			}
		}
	}

	return err

}

func (app *CommonApp) generateDbWatchKeys(d *db.DB, isDeleteOp bool) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys

	return keys, err
}
