/*
 * Copyright (c) 2002-2018 "Neo4j,"
 * Neo4j Sweden AB [http://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package neo4j

import (
	"net/url"
	"sync/atomic"

	"github.com/neo4j-drivers/neo4j-go-connector"
	"github.com/pkg/errors"
)

type directDriver struct {
	config    *Config
	target    url.URL
	connector seabolt.Connector

	open int32
}

func configToConnectorConfig(config *Config) *seabolt.Config {
	return &seabolt.Config{
		Encryption:  config.Encrypted,
		Debug:       config.Log.DebugEnabled(),
		MaxPoolSize: config.MaxConnectionPoolSize,
	}
}

func newDirectDriver(target *url.URL, token AuthToken, config *Config) (*directDriver, error) {
	if config == nil {
		config = defaultConfig()
	}

	connector, err := seabolt.NewConnector(target.String(), token.tokens, configToConnectorConfig(config))
	if err != nil {
		return nil, err
	}

	driver := directDriver{
		config:    config,
		target:    *target,
		connector: connector,
		open:      1,
	}
	return &driver, nil
}

func assertDriverOpen(driver *directDriver) error {
	if atomic.LoadInt32(&driver.open) == 0 {
		return errors.New("cannot acquire a session on a closed driver")
	}

	return nil
}

func (driver *directDriver) Target() url.URL {
	return driver.target
}

func (driver *directDriver) Session(accessMode AccessMode, bookmarks ...string) (*Session, error) {
	if err := assertDriverOpen(driver); err != nil {
		return nil, err
	}

	return newSession(driver, accessMode, bookmarks), nil
}

func (driver *directDriver) Close() error {
	if atomic.CompareAndSwapInt32(&driver.open, 1, 0) {
		return driver.connector.Close()
	}

	return nil
}

func (driver *directDriver) configuration() *Config {
	return driver.config
}

func (driver *directDriver) acquire(mode AccessMode) (seabolt.Connection, error) {
	if err := assertDriverOpen(driver); err != nil {
		return nil, err
	}

	pool, err := driver.connector.GetPool()
	if err != nil {
		return nil, err
	}

	return pool.Acquire()
}
