package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// 读取历史数据（红球按顺序，蓝球单独）
func readHistoryData(filename string) ([][]int, []int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	var redHistory [][]int // 红球历史（每一行是一期的6个红球，按顺序）
	var blueHistory []int  // 蓝球历史

	for _, record := range records {
		// 解析红球（前6列）
		red := make([]int, 6)
		for i := 0; i < 6; i++ {
			r, err := strconv.Atoi(record[i])
			if err != nil {
				return nil, nil, err
			}
			red[i] = r
		}
		// 解析蓝球（第7列）
		blue, err := strconv.Atoi(record[6])
		if err != nil {
			return nil, nil, err
		}
		redHistory = append(redHistory, red)
		blueHistory = append(blueHistory, blue)
	}

	return redHistory, blueHistory, nil
}

// Contains 函数用于判断目标字符串是否在字符串切片中
func Contains(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

// 读取历史数据（红球按顺序，蓝球单独）
func readHistoryDataFromSql(filename string) ([][]int, []int, error) {
	absDir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取程序工作目录失败，错误描述：" + err.Error())
		return nil, nil, err
	}
	db, err := sql.Open("sqlite3", absDir+"/"+filename)
	if err != nil {
		fmt.Printf("sqlite open failed:[%v]", err.Error())
		return nil, nil, err
	}
	defer db.Close()

	querySql := "select id, lottery, create_time,code, date, red, blue, my_prize_grade from lottery order by create_time;"

	//fmt.Println("querySql:", querySql)
	stmt, err := db.Prepare(querySql)
	if err != nil {
		fmt.Println("Prepare error:", err)
		return nil, nil, err
	}
	rows, err := stmt.Query()
	if err != nil {
		fmt.Println("query error:", err)
		return nil, nil, err
	}
	defer rows.Close()

	var distinctRed []string
	var redHistory [][]int // 红球历史（每一行是一期的6个红球，按顺序）
	var blueHistory []int  // 蓝球历史

	for rows.Next() {
		var item Lotterys
		err = rows.Scan(&item.Id, &item.Lottery, &item.CreateTime, &item.Code, &item.Date, &item.Red, &item.Blue, &item.MyPrizeGrade)
		if err != nil {
			fmt.Println("Scan error:", err)
			continue
		}

		if item.Red.Valid {
			//去重
			if Contains(distinctRed, item.Red.String) {
				continue
			}
			distinctRed = append(distinctRed, item.Red.String)

			// fmt.Println(item.Red.String)
			record := strings.Split(item.Red.String, ",")

			red := make([]int, 6)
			for i := 0; i < 6; i++ {
				r, err := strconv.Atoi(record[i])
				if err != nil {
					return nil, nil, err
				}
				red[i] = r
			}
			redHistory = append(redHistory, red)
		}
		if item.Blue.Valid {
			// 解析蓝球（第7列）
			blue, err := strconv.Atoi(item.Blue.String)
			if err != nil {
				return nil, nil, err
			}
			blueHistory = append(blueHistory, blue)
		}
	}

	return redHistory, blueHistory, nil
}

// 构建红球一阶马尔可夫链转移概率表
func buildRedTransition(redHistory [][]int) map[int]map[int]float64 {
	// 统计转移次数：count[当前状态][下一个状态] = 次数
	count := make(map[int]map[int]int)

	// 处理起始状态（0→第一个球）
	for _, reds := range redHistory {
		if len(reds) == 0 {
			continue
		}
		first := reds[0]
		if _, ok := count[0]; !ok {
			count[0] = make(map[int]int)
		}
		count[0][first]++
	}

	// 处理后续状态（第n个球→第n+1个球）
	for _, reds := range redHistory {
		for i := 0; i < len(reds)-1; i++ {
			current := reds[i]
			next := reds[i+1]
			if _, ok := count[current]; !ok {
				count[current] = make(map[int]int)
			}
			count[current][next]++
		}
	}

	// 归一化：将次数转换为概率
	prob := make(map[int]map[int]float64)
	for current, nextCounts := range count {
		total := 0
		for _, c := range nextCounts {
			total += c
		}
		if total == 0 {
			continue
		}
		prob[current] = make(map[int]float64)
		for next, c := range nextCounts {
			prob[current][next] = float64(c) / float64(total)
		}
	}

	return prob
}

// 构建蓝球频率概率表（统计历史出现次数）
func buildBlueProbability(blueHistory []int) map[int]float64 {
	count := make(map[int]int)
	total := 0
	for _, b := range blueHistory {
		count[b]++
		total++
	}

	// 归一化：次数→概率
	prob := make(map[int]float64)
	for b, c := range count {
		prob[b] = float64(c) / float64(total)
	}

	return prob
}

// 生成6个不重复的红球（基于马尔可夫链转移概率）
func generateRedNumbers(transition map[int]map[int]float64) []int {
	rand.Seed(time.Now().UnixNano())
	var result []int
	selected := make(map[int]bool) // 已选红球（避免重复）
	currentState := 0              // 起始状态

	for i := 0; i < 6; i++ {
		// 获取当前状态的转移概率表
		nextProbs, ok := transition[currentState]
		if !ok || len(nextProbs) == 0 {
			//  fallback：无历史数据时，随机选未被选中的红球
			next := randomUnselected(selected)
			result = append(result, next)
			selected[next] = true
			currentState = next
			continue
		}

		// 过滤已选红球（避免重复）
		filteredProbs := make(map[int]float64)
		totalProb := 0.0
		for next, p := range nextProbs {
			if !selected[next] {
				filteredProbs[next] = p
				totalProb += p
			}
		}

		// 归一化过滤后的概率（确保总和为1）
		normalizedProbs := make(map[int]float64)
		for next, p := range filteredProbs {
			normalizedProbs[next] = p / totalProb
		}

		// 轮盘赌法：根据概率选择下一个红球
		next := selectByProbability(normalizedProbs)
		result = append(result, next)
		selected[next] = true
		currentState = next // 更新状态为当前选中的红球
	}

	return result
}

// 随机选一个未被选中的红球（1-33）
func randomUnselected(selected map[int]bool) int {
	for {
		num := rand.Intn(33) + 1
		if !selected[num] {
			return num
		}
	}
}

// 轮盘赌法：根据概率选择元素（如probs={5:0.67, 8:0.33}，随机选5的概率更高）
func selectByProbability(probs map[int]float64) int {
	r := rand.Float64()
	var sum float64
	for num, p := range probs {
		sum += p
		if sum >= r {
			return num
		}
	}
	//  fallback：返回第一个元素（理论上不会触发）
	for num := range probs {
		return num
	}
	return 0
}

// 生成1个蓝球（基于历史频率）
func generateBlueNumber(probs map[int]float64) int {
	if len(probs) == 0 {
		//  fallback：无历史数据时，随机选1-16
		return rand.Intn(16) + 1
	}
	return selectByProbability(probs) // 复用轮盘赌法
}
