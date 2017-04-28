package db

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"usercenter/user"

	"gopkg.in/pg.v3"
)

type PostgresDBConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`

	User     string `json:"user"`
	Password string `json:"passwd"`

	Database string `json:"database"`

	DialTimeout  int `json:"dial_timeout"`
	ReadTimeout  int `json:"read_timeout"`
	WriteTimeout int `json:"write_timeout"`
}

func NewPostgresDBConfig(configFile string) *PostgresDBConfig {
	file, err := os.Open(configFile)
	if err != nil {
		log.Printf("open file get error:%s", err.Error())
		return nil
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("read file get error:%s", err.Error())
		return nil
	}

	dbConfig := &PostgresDBConfig{}
	err = json.Unmarshal(data, dbConfig)
	if err != nil {
		log.Printf("unmarshal config file get error:%s", err.Error())
		return nil
	}
	return dbConfig
}

type PostgresQlDb struct {
	dbOptions *pg.Options
	db        *pg.DB
}

func NewPostgresQlDb(configFile string) *PostgresQlDb {
	config := NewPostgresDBConfig(configFile)
	if config == nil {
		panic("failed new config")
	}

	return NewPostgredQlDbWithConfig(config)
}

func NewPostgredQlDbWithConfig(config *PostgresDBConfig) *PostgresQlDb {
	ret := &PostgresQlDb{
		dbOptions: &pg.Options{
			Host:         config.Host,
			Port:         config.Port,
			User:         config.User,
			Password:     config.Password,
			Database:     config.Database,
			DialTimeout:  time.Duration(config.DialTimeout) * time.Second,
			ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
		},
	}
	ret.db = pg.Connect(ret.dbOptions)
	if ret.db == nil {
		panic("failed to connect db")
	}
	return ret
}

func (self *PostgresQlDb) AddUser(users []*user.User) error {
	// ToDo(batch insert)
	for _, user := range users {
		_, err := self.db.ExecOne(`INSERT INTO users
			(id, name, description, gender, age, createdtime, address, type)
			 VALUES (?id, ?name, ?description, ?gender, ?age, ?createdtime, ?address, ?type)`, user)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *PostgresQlDb) GetUser(name string) *user.User {
	user := &user.User{}
	_, err := self.db.QueryOne(&user,
		`SELECT id, name, description, gender, age, createdtime, address, type FROM users WHERE name=?`,
		name)
	if err != nil {
		return nil
	}
	return user
}

func (self *PostgresQlDb) LoadUserList() ([]*user.User, error) {
	userList := []*user.User{}
	_, err := self.db.Query(&userList,
		`SELECT id, name, description, gender, age, createdtime, address, type FROM users`)
	if err != nil {
		return nil, err
	}
	return userList, nil
}

func (self *PostgresQlDb) UpdateUserRelations(
	relations []*user.UserRelationShip) error {
	for _, relation := range relations {
		_, err := self.UpdateUserRelation(relation)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *PostgresQlDb) GetRelationWithOtherUserId(userId,
	otherUserId int64) *user.UserRelationShip {
	relation := &user.UserRelationShip{}
	_, err := self.db.QueryOne(relation,
		`SELECT user_id, state, otherside FROM relations 
		WHERE user_id=? AND otherside=?`, userId, otherUserId)
	if err != nil {
		return nil
	}
	return relation
}

func (self *PostgresQlDb) UpdateUserRelation(
	relation *user.UserRelationShip) (*user.UserRelationShip, error) {

	otherSideRelation := self.GetRelationWithOtherUserId(relation.OtherSide, relation.Id)
	if relation.State == user.RELATION_STATE_LICKED &&
		otherSideRelation != nil &&
		otherSideRelation.State == user.RELATION_STATE_LICKED {
		relation.State = user.RELATION_STATE_MATCHED
		_, err := self.db.ExecOne(
			`UPDATE relations SET state=? WHERE user_id=? AND otherside=?`,
			user.RELATION_STATE_MATCHED, relation.OtherSide, relation.Id)
		if err != nil {
			return nil, err
		}
		_, err = self.db.ExecOne(`INSERT INTO relations
			(user_id, state, otherside)
			 VALUES (?id, ?state, ?otherside)`, relation)
		if err != nil {
			return nil, err
		}
		return relation, nil
	}

	_, err := self.db.ExecOne(`INSERT INTO relations
			(user_id, state, otherside)
			VALUES(?user_id, ?state, ?otherside)`, relation)
	if err != nil {
		return nil, err
	}
	return relation, nil
}

func (self *PostgresQlDb) GetUserRelation(
	userId int64) ([]*user.UserRelationShip, error) {
	relations := []*user.UserRelationShip{}
	_, err := self.db.Query(&relations,
		`SELECT user_id, state, otherside FROM relations where user_id=?`,
		userId)
	if err != nil {
		return nil, err
	}
	return relations, nil
}

func (self *PostgresQlDb) Close() error {
	if self.db != nil {
		return self.db.Close()
	}
	return nil
}
