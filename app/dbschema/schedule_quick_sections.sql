-- 行程「新建事项」弹窗：一级分组 + 二级选项（条目来自 location_catalog）
CREATE TABLE IF NOT EXISTS schedule_quick_section (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  section_label TEXT NOT NULL,
  item_category TEXT NOT NULL,
  item_region TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_schedule_quick_section_sort ON schedule_quick_section (sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS ux_schedule_quick_section_natural ON schedule_quick_section (section_label, item_category, item_region);

INSERT OR IGNORE INTO schedule_quick_section (sort_order, section_label, item_category, item_region) VALUES
(1, '在岗办公', 'hotel_intro', ''),
(2, '商务用餐 · 半岛店餐厅', 'dining_restaurant', 'peninsula'),
(3, '商务用餐 · 氹仔店餐厅', 'dining_restaurant', 'cotai'),
(4, '快捷事项', 'schedule_chip', '');
