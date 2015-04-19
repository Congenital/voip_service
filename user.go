/**
 * Copyright (c) 2014-2015, GoBelieve     
 * All rights reserved.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
 */

package main
import "math/rand"
import "fmt"
import "time"
import log "github.com/golang/glog"
import "github.com/garyburd/redigo/redis"
import "errors"

const CHARACTER_SET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenUserToken() string {
	b := make([]byte, 30)
	for i := 0; i < 30; i++ {
		r := rand.Int()%len(CHARACTER_SET)
		b[i] = CHARACTER_SET[r]
	}
	return string(b)
}

func GetUserAccessToken(appid int64, uid int64) string {
	conn := redis_pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("users_%d_%d", appid, uid)
	token, err := redis.String(conn.Do("HGET", key, "access_token"))
	if err != nil {
		log.Infof("hget %s err:%s\n", key, err)
		return ""
	}
	return token
}

func LoadUserAccessToken(token string) (int64, int64, string, error) {
	conn := redis_pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("access_token_%s", token)
	var uid int64
	var appid int64
	var uname string

	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return 0, 0, "", err
	}
	if !exists {
		return 0, 0, "", errors.New("token non exists")
	}

	reply, err := redis.Values(conn.Do("HMGET", key, "user_id", "app_id", "user_name"))
	if err != nil {
		log.Info("hmget error:", err)
		return 0, 0, "", err
	}

	_, err = redis.Scan(reply, &uid, &appid, &uname)
	if err != nil {
		log.Warning("scan error:", err)
		return 0, 0, "", err
	}
	return appid, uid, uname, nil	
}

func SaveUserAccessToken(appid int64, uid int64, uname string, token string) error {
	conn := redis_pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("access_token_%s", token)
	
	_, err := conn.Do("HMSET", key, "user_id", uid, "user_name", uname, "app_id", appid)
	if err != nil {
		log.Info("hmset err:", err)
		return err
	}

	key = fmt.Sprintf("users_%d_%d", appid, uid)
	_, err = conn.Do("HSET", key, "access_token", token)
	if err != nil {
		log.Info("hget err:", err)
		return err
	}
	return nil
}

func SaveUserDeviceToken(appid int64, uid int64, device_token string, ng_device_token string) error {
	conn := redis_pool.Get()
	defer conn.Close()

	now := time.Now().Unix()
	key := fmt.Sprintf("users_%d_%d", appid, uid)
	if len(device_token) > 0 {
		_, err := conn.Do("HMSET", key, "apns_device_token", device_token, 
			"apns_timestamp", now)
		if err != nil {
			log.Info("hget err:", err)
			return err
		}
	}
	if len(ng_device_token) > 0 {
		_, err := conn.Do("HMSET", key, "ng_device_token", ng_device_token, 
			"ng_timestamp", now)
		if err != nil {
			log.Info("hget err:", err)
			return err
		}
	}
	return nil	
}

func ResetUserDeviceToken(appid int64, uid int64, device_token string, ng_device_token string) error {
	conn := redis_pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("users_%d_%d", appid, uid)
	if len(device_token) > 0 {
		token, err := redis.String(conn.Do("HGET", key, "apns_device_token"))
		if err != nil {
			log.Info("hget err:", err)
			return err
		}
		if token != device_token {
			log.Infof("reset apns token:%s device token:%s\n", token, device_token)
			return nil
		}
		_, err = conn.Do("HDEL", key, "apns_device_token", "apns_timestamp")
		if err != nil {
			log.Info("hdel err:", err)
			return err
		}
	}

	if len(ng_device_token) > 0 {
		token, err := redis.String(conn.Do("HGET", key, "ng_device_token"))
		if err != nil {
			log.Info("hget err:", err)
			return err
		}
		if token != ng_device_token {
			log.Infof("reset ng token:%s device token:%s\n", token, ng_device_token)
			return nil
		}
		_, err = conn.Do("HDEL", key, "ng_device_token", "ng_timestamp")
		if err != nil {
			log.Info("hdel err:", err)
			return err
		}
	}
	return nil	
}
