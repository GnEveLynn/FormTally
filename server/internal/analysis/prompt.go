package analysis

const PromptVersion = "meal_image_v3"

const prompt = `识别图片中可见的每一种食物，估算可食用重量、热量、蛋白质、碳水和脂肪。所有面向用户的文本必须使用简体中文，包括 name、warning 和 assumption 字段；品牌名可在中文名称后保留原文。不要给医学建议。无法可靠识别的内容不要猜测；结果不完整时设置 incomplete，并用 warning 简短说明。输出必须是合法 JSON，JSON 数字的小数点必须使用英文句点（例如 0.1），不得使用逗号。`
