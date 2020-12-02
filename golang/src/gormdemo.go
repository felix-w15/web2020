package main

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net/url"
	"strconv"
)

//request format
/*
	myStuNo
	buildingNo
	numOfStu
	roommatesStuNo
*/

//Building ...
type Building struct {
	ID           int    `gorm:"primaryKey"`
	BuildingNo   int    `gorm:"column:buildingNo"`
	BuildingName string `gorm:"column:buildingName"`
	Status       int    `gorm:"column:status"` //0 新生不可见
}

//APIStudent ...
type APIStudent struct {
	ID           int    `gorm:"primaryKey"`
	StudentNo    string `gorm:"column:studentNo"`
	Gender       int
	Year         int
	Roomstatus   int
	Onlinestatus int
}

//Bed
type Bed struct {
	ID            int `gorm:"primaryKey"`
	RoomId        int
	BedNo         string `gorm:"column:bedNo"`
	IsDistributed int
	StudentId     int
	status        int
}

//Room
type Room struct {
	ID          int `gorm:"primaryKey"`
	BuildingId  int
	Gender      int
	RoomName    string `gorm:"column:roomName"`
	Status      int
	EmptyBedNum int `gorm:"column:emptyBedNum"`
	TotalBedNum int `gorm:"column:totalBedNum"`
}

func handleDormRequ(form url.Values) (int, string) {
	// dsn := "web2020:123456@tcp(127.0.0.1:3306)/web2020?charset=utf8mb4&parseTime=True&loc=Local"
	//
	fmt.Println(form)
	buildingNo := form["buildingNo"][0]
	numOfStu, _ := strconv.Atoi(form["numOfStu"][0])
	studentNo := form["myStuNo"][0]
	roommatesStuNo := form["roommatesStuNo"]

	//连接数据库

	// db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	// if err != nil {
	// 	fmt.Println(err)
	// }

	//返回值code 以及data
	code := 0
	data := ""

	// 取出楼ID
	building := Building{}
	errBuilding := Db.Table("web2020_dorm_building").Where(" status = 1").
		Where(" buildingNO = ?", buildingNo).First(&building).Error
	if errors.Is(errBuilding, gorm.ErrRecordNotFound) {
		fmt.Println("该楼不满足新生住宿申请条件")
		code = 404
		data = "该楼不满足新生住宿申请条件"
		return code, data
	} else {
		fmt.Println(building.BuildingName)
	}

	// 检查人员是否符合申请条件，必须都为新生，并且在名单内，性别与申请的一致,检查人员是否已经分配了宿舍
	students := make([]APIStudent, numOfStu)
	errStu := Db.Table("web2020_student").Where("studentNo = ?", studentNo).
		Where("year = 2020 AND roomstatus = 0 AND onlinestatus = 1").First(&(students[0])).Error
	if errors.Is(errStu, gorm.ErrRecordNotFound) {
		fmt.Println("该生不满足新生住宿申请条件")
		code = 404
		data = "该生不满足新生住宿申请条件"
		return code, data
	} else {
		fmt.Println(students[0].StudentNo)
	}
	//检查性别与入学年份
	shouldGender := students[0].Gender
	for i := 0; i < len(roommatesStuNo); i++ {
		tempErrStu := Db.Table("web2020_student").Where("studentNo = ?", roommatesStuNo[i]).
			Where("gender = ?", shouldGender).Where("year = 2020 AND roomstatus = 0 AND onlinestatus = 1").
			First(&(students[i+1])).Error
		if errors.Is(tempErrStu, gorm.ErrRecordNotFound) {
			fmt.Println("该生不满足新生住宿申请条件")
			code = 404
			data = "学生之一不满足新生住宿申请条件或性别不一致"
			return code, data
		} else {
			fmt.Println(students[i+1].StudentNo)
		}
	}

	//for i := 0; i < len(students); i++{
	//	fmt.Println(students[i])
	//}



	//循环标志
	flag2 := true

	for flag2 {

		// 如果找到
		// 1、写入宿舍数据
		// 将学号，姓名  写入到相应的宿舍床位中
		// (1)取出该房间内的床位个数，床位状态必须为可用，未分配，
		// (2)检查数量的否符合分配的数量
		// (3)分别写入相应的床位状态，已分配、学号、姓名，将学生的状态设置已经分配宿舍
		// 2、将房间的空床数设置为相应的真实值
		// 将宿舍空床位数减去入住人数
		// 3、把申请宿舍订单写入订单表中,状态为成功

		//此次事务是否回滚标志
		ifRollback := false

		tx := Db.Begin()

		//房间
		room := Room{}
		errorRoom := Db.Table("web2020_dorm_room").Where("building_id = ?", building.ID).
			Where("gender = ?", shouldGender).Where("status = 1").
			Where("emptyBedNum >= ?", numOfStu).First(&room).Error
		if errors.Is(errorRoom, gorm.ErrRecordNotFound) {
			fmt.Println("没有符合申请条件的房间")
			code = 404
			data = "没有符合申请条件的房间"
			return code, data
		}
		//写入宿舍数据
		//学号，姓名写入宿舍床位中
		room.EmptyBedNum = room.EmptyBedNum - numOfStu
		//占领房间
		error1 := Db.Table("web2020_dorm_room").Save(&room).Error
		if error1 != nil {
			ifRollback = true
		}
		//占领床位
		var beds []Bed
		error2 := Db.Table("web2020_dorm_bed").Where("room_id = ?", room.ID).
			Where("is_distributed = 0").Where("status = 1").Find(&beds).Error
		if error2 != nil {
			ifRollback = true
		}
		if len(beds) < numOfStu {
			ifRollback = true
		}
		for i := 0; !ifRollback && i < numOfStu; i++ {
			beds[i].IsDistributed = 1
			beds[i].StudentId = students[i].ID
			tempErro := Db.Table("web2020_dorm_bed").Save(&(beds[i])).Error
			if tempErro != nil {
				ifRollback = true
			}
		}
		//更新学生状态
		for i := 0; !ifRollback && i < len(students); i++ {
			students[i].Roomstatus = 1
			tempError := Db.Table("web2020_student").Save(&(students[i])).Error
			if tempError != nil {
				ifRollback = true
			}
		}
		if ifRollback == true {
			//回滚
			tx.Rollback()
		} else { // 否则，提交事务
			tx.Commit()
			fmt.Println("符合条件，申请成功")
			return 200, "申请成功宿舍号为" + room.RoomName
		}

	}
	return 404, "没有符合条件的宿舍"

}
