-- MGM location & dining catalog (editable via SQLite)
CREATE TABLE IF NOT EXISTS location_catalog (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  category TEXT NOT NULL,
  region TEXT,
  title TEXT NOT NULL,
  subtitle TEXT,
  detail TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  sync_uid TEXT UNIQUE
);
CREATE INDEX IF NOT EXISTS idx_location_catalog_category ON location_catalog(category);
CREATE INDEX IF NOT EXISTS idx_location_catalog_region ON location_catalog(region);

INSERT OR IGNORE INTO location_catalog (slug, category, region, title, subtitle, detail, sort_order) VALUES
('mgm_peninsula_hotel','hotel_intro','peninsula','澳门美高梅（半岛店）','澳门半岛外港新填海区 · 2007 开业 · 福布斯五星','【澳门美高梅（半岛店）】
位置：澳门半岛外港新填海区孙逸仙大马路
开业：2007 年
定位：经典欧式奢华综合度假酒店，连续 11 年获《福布斯旅游指南》五星评级

酒店介绍
澳门美高梅是澳门老牌奢华酒店的代表，其波浪形玻璃幕墙由黄金色、白金色、玫瑰金色三种玻璃铺设而成，是澳门半岛的标志性建筑之一。酒店高 35 层，拥有约 600 间豪华客房及套房。
酒店的核心是天幕广场，25 米高的巨型玻璃天幕复刻了里斯本火车站的浪漫，融合了葡萄牙传统建筑风格与南欧风情，是热门拍照打卡点。二楼设有保利美高梅博物馆，面积近 2000 平方米，按照国家一级文物展陈标准建造，常年展出跨越中西文明的珍贵文物。

主要餐厅
金殿堂：米其林推荐餐厅，主打手工粤菜，糅合经典与现代岭南菜系精髓，代表菜品有岭南脆皮渔香百花鸡、金腿燕窝鹧鸪粥等
话匣子：2026 年新开业的新加坡知名餐饮品牌，以现代雅致空间呈现狮城精神，招牌菜包括新加坡文华鸡饭、龙虾喇沙等
宝雅座：法式餐厅兼开放式酒吧，拥有澳门唯一的双层酒窖，可做私宴，酒单屡获殊荣
食・八方：包含北厨和南苑两个区域，北厨提供北方及全国各地佳肴，南苑为广东茶餐厅概念

主要设施
禅潺水疗中心：连续 7 年获《福布斯旅游指南》五星殊荣，提供「禅潺流韵」水中指压按摩、喜马拉雅盐石按摩等特色项目
室外游泳池：环境优雅，设有按摩池
24 小时健身中心：配备顶级健身设备
会议及宴会设施：功能齐全，是举办各类宴会的理想地点
精品零售商店：汇集多个国际知名品牌',1),
('mgm_cotai_hotel','hotel_intro','cotai','美狮美高梅（氹仔店）','路凼金光大道 · 2018 开业 · 福布斯五星','【美狮美高梅（氹仔店）】
位置：澳门路凼城金光大道
开业：2018 年
定位：现代科技与艺术融合的豪华综合度假酒店，连续 4 年获《福布斯旅游指南》五星评级

酒店介绍
美狮美高梅由全球知名建筑事务所 KPF 操刀设计，外观如同九个错落堆叠的珍宝盒，呈现出雕塑般的现代气息。酒店入口处矗立着 11 米高、38 吨重的 24K 金箔狮子雕塑，是何超琼亲自揭幕的品牌精神象征。
酒店的核心是视博广场，其天幕面积达 8100 平方米，荣获吉尼斯世界纪录「最大悬跨网架式玻璃屋顶」。广场设有 LED 天幕空中海洋秀，每天上演精彩的数码艺术表演。酒店共拥有约 1400 间融合高雅与舒适的客房及套房，由美狮主楼、美艺酒店、天乐阁、御狮别墅及雍华府五大住宿区组成。

主要餐厅
蜀道：连续 4 年获米其林一星，主打高级川菜，调料均从四川空运而来，推荐芙蓉官燕鸡豆花和老成都灯影鱼片
雅吉：连续 2 年获米其林一星，以亚洲料理酒馆概念为核心，以法式料理技艺创意呈现多元亚洲风味
淳餐厅：由北京利苑创始总厨主理，提供地道粤菜，招牌菜有脆皮龙井茶熏鸡、香酥云南白菌焗酿蟹盖等
涛岸：以海岸生活的悠闲节奏为灵感，呈献国际佳肴与澳葡特色料理

主要设施
美高梅剧院：亚洲首个多元化动感剧院，上演由张艺谋导演与何超琼联手打造的驻场巨作《澳门 2049》
禅潺水疗中心：连续 7 年获《福布斯旅游指南》五星殊荣，提供私密理疗空间和专业按摩服务
室外游泳池：以玻璃天幕为背景，设有按摩区，周围绿植环绕
24 小时健身中心：配备全套 Technogym 系列顶级健身设备
会议及宴会设施：可容纳千人的宴会厅及多个多功能会议室
精品零售商店：汇集多个国际奢侈品牌',2),
('dining_peninsula_jindiantang','dining_restaurant','peninsula','金殿堂','米其林推荐 · 手工粤菜','米其林推荐餐厅，主打手工粤菜，糅合经典与现代岭南菜系精髓，代表菜品有岭南脆皮渔香百花鸡、金腿燕窝鹧鸪粥等。',10),
('dining_peninsula_wahagizi','dining_restaurant','peninsula','话匣子','新加坡风味 · 2026 新店','新加坡知名餐饮品牌，现代雅致空间呈现狮城精神，招牌包括新加坡文华鸡饭、龙虾喇沙等。',11),
('dining_peninsula_boya','dining_restaurant','peninsula','宝雅座','法式 · 双层酒窖','法式餐厅兼开放式酒吧，澳门唯一的双层酒窖，可做私宴，酒单屡获殊荣。',12),
('dining_peninsula_shibafang','dining_restaurant','peninsula','食・八方','北厨 · 南苑','包含北厨和南苑：北厨北方及全国各地佳肴，南苑广东茶餐厅概念。',13),
('dining_cotai_shudao','dining_restaurant','cotai','蜀道','米其林一星 · 川菜','连续 4 年米其林一星，高级川菜，调料四川空运，推荐芙蓉官燕鸡豆花、老成都灯影鱼片。',20),
('dining_cotai_yaji','dining_restaurant','cotai','雅吉','米其林一星 · 亚洲酒馆','亚洲料理酒馆概念，法式技艺呈现多元亚洲风味。',21),
('dining_cotai_chun','dining_restaurant','cotai','淳餐厅','利苑创始总厨 · 粤菜','地道粤菜，脆皮龙井茶熏鸡、香酥云南白菌焗酿蟹盖等。',22),
('dining_cotai_taoan','dining_restaurant','cotai','涛岸','海岸灵感 · 国际澳葡','国际佳肴与澳葡特色料理。',23),
('chip_client_visit','schedule_chip','','客户拜访','','',30),
('chip_meeting','schedule_chip','','会议','','',31),
('chip_docs','schedule_chip','','文档整理','','',32);