package v1alpha1

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/layer5io/meshkit/database"
	"github.com/layer5io/meshkit/models/meshmodel/core/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TypeMeta struct {
	Kind       string `json:"kind,omitempty" yaml:"kind"`
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion"`
}
type ComponentFormat string

const (
	JSON ComponentFormat = "JSON"
	YAML ComponentFormat = "YAML"
	CUE  ComponentFormat = "CUE"
)

// swagger:response ComponentDefinition
// use NewComponent function for instantiating
type ComponentDefinition struct {
	ID uuid.UUID `json:"-"`
	TypeMeta
	DisplayName string                 `json:"displayName" gorm:"displayName"`
	Format      ComponentFormat        `json:"format" yaml:"format"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
	Model       Model                  `json:"model"`
	Schema      string                 `json:"schema" yaml:"schema"`
	CreatedAt   time.Time              `json:"-"`
	UpdatedAt   time.Time              `json:"-"`
}
type ComponentDefinitionDB struct {
	ID      uuid.UUID `json:"-"`
	ModelID uuid.UUID `json:"-" gorm:"modelID"`
	TypeMeta
	DisplayName string          `json:"displayName" gorm:"displayName"`
	Format      ComponentFormat `json:"format" yaml:"format"`
	Metadata    []byte          `json:"metadata" yaml:"metadata"`
	Schema      string          `json:"schema" yaml:"schema"`
	CreatedAt   time.Time       `json:"-"`
	UpdatedAt   time.Time       `json:"-"`
}

func (c ComponentDefinition) Type() types.CapabilityType {
	return types.ComponentDefinition
}
func (c ComponentDefinition) Doc(f DocFormat, db *database.Handler) (doc string) {
	switch f {
	case HTMLFormat:
		data := fmt.Sprintf("%s supports following relationships: ", c.Kind)
		//Todo: Scan registry to get relationships for the given c.Kind and c.Model
		data += fmt.Sprintf("\n%s supports following policies: ", c.Kind)
		//Todo: Scan registry to get policies for the given c.Kind and c.Model
		doc = fmt.Sprintf(`
		<html>
		%s
		</html>
		`, data)
	}
	return doc
}

func (c ComponentDefinition) GetID() uuid.UUID {
	return c.ID
}

func CreateComponent(db *database.Handler, c ComponentDefinition) (uuid.UUID, error) {
	c.ID = uuid.New()
	tempModelID := uuid.New()
	byt, err := json.Marshal(c.Model)
	if err != nil {
		return uuid.UUID{}, err
	}
	modelID := uuid.NewSHA1(uuid.UUID{}, byt)
	var model Model
	modelCreationLock.Lock()
	err = db.First(&model, "id = ?", modelID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return uuid.UUID{}, err
	}
	if model.ID == tempModelID || err == gorm.ErrRecordNotFound { //The model is already not present and needs to be inserted
		model = c.Model
		model.ID = modelID
		err = db.Create(&model).Error
		if err != nil {
			modelCreationLock.Unlock()
			return uuid.UUID{}, err
		}
	}
	modelCreationLock.Unlock()
	cdb := c.GetComponentDefinitionDB()
	cdb.ModelID = model.ID
	err = db.Create(&cdb).Error
	return c.ID, err
}
func GetMeshModelComponents(db *database.Handler, f ComponentFilter) (c []ComponentDefinition) {
	type componentDefinitionWithModel struct {
		ComponentDefinitionDB
		Model
	}

	var componentDefinitionsWithModel []componentDefinitionWithModel
	finder := db.Model(&ComponentDefinitionDB{}).
		Select("component_definition_dbs.*, models.*").
		Joins("JOIN models ON component_definition_dbs.model_id = models.id") //
	if f.Name != "" {
		if f.Greedy {
			finder = finder.Where("component_definition_dbs.kind LIKE ?", f.Name+"%")
		} else {
			finder = finder.Where("component_definition_dbs.kind = ?", f.Name)
		}
	}
	if f.APIVersion != "" {
		finder = finder.Where("component_definition_dbs.api_version = ?", f.APIVersion)
	}
	if f.ModelName != "" {
		finder = finder.Where("models.name = ?", f.ModelName)
	}
	if f.Version != "" {
		finder = finder.Where("models.version = ?", f.Version)
	}
	if f.OrderOn != "" {
		if f.Sort == "desc" {
			finder = finder.Order(clause.OrderByColumn{Column: clause.Column{Name: f.OrderOn}, Desc: true})
		} else {
			finder = finder.Order(f.OrderOn)
		}
	}
	finder = finder.Offset(f.Offset)
	if f.Limit != 0 {
		finder = finder.Limit(f.Limit)
	}
	err := finder.
		Scan(&componentDefinitionsWithModel).Error
	if err != nil {
		fmt.Println(err.Error()) //for debugging
	}
	for _, cm := range componentDefinitionsWithModel {
		c = append(c, cm.ComponentDefinitionDB.GetComponentDefinition(cm.Model))
	}
	return c
}

type ComponentFilter struct {
	Name       string
	APIVersion string
	Greedy     bool //when set to true - instead of an exact match, name will be prefix matched
	ModelName  string
	Version    string
	Sort       string //asc or desc. Default behavior is asc
	OrderOn    string
	Limit      int //If 0 or  unspecified then all records are returned and limit is not used
	Offset     int
}

// Create the filter from map[string]interface{}
func (cf *ComponentFilter) Create(m map[string]interface{}) {
	if m == nil {
		return
	}
	cf.Name = m["name"].(string)
}

func (cmd *ComponentDefinitionDB) GetComponentDefinition(model Model) (c ComponentDefinition) {
	c.ID = cmd.ID
	c.TypeMeta = cmd.TypeMeta
	c.Format = cmd.Format
	c.DisplayName = cmd.DisplayName
	if c.Metadata == nil {
		c.Metadata = make(map[string]interface{})
	}
	_ = json.Unmarshal(cmd.Metadata, &c.Metadata)
	c.Schema = cmd.Schema
	c.Model = model
	return
}
func (c *ComponentDefinition) GetComponentDefinitionDB() (cmd ComponentDefinitionDB) {
	cmd.ID = c.ID
	cmd.TypeMeta = c.TypeMeta
	cmd.Format = c.Format
	cmd.Metadata, _ = json.Marshal(c.Metadata)
	cmd.DisplayName = c.DisplayName
	cmd.Schema = c.Schema
	return
}
