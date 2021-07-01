package migrations

import (
	"github.com/krixano/ponixserver/src/migration/types"
	"github.com/krixano/ponixserver/src/db"
)

var All map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)
var Music map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)
var LifeKept map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)
var StarWars map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)
var Search map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)

func GetMap(database db.DBType) map[types.MigrationVersion]types.Migration {
	if database == db.MusicDB {
		return Music
	} else if database == db.LifeKeptDB {
		return LifeKept
	} else if database == db.StarWarsDB {
		return StarWars
	} else if database == db.SearchDB {
		return Search
	}

	return Music
}

func registerMigration(m types.Migration) {
	All[m.Version()] = m

	database := m.DB()
	if database == db.MusicDB {
		Music[m.Version()] = m
	} else if database == db.LifeKeptDB {
		LifeKept[m.Version()] = m
	} else if database == db.StarWarsDB {
		StarWars[m.Version()] = m
	} else if database == db.SearchDB {
		Search[m.Version()] = m
	}
}
