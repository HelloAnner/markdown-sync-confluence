import os
import yaml

class ConfigLoader:
    """配置加载器，处理配置文件、环境变量和命令行参数"""

    @staticmethod
    def load_config(config_path=None, cli_config=None):
        """加载配置，优先级：命令行参数 > 环境变量 > 配置文件"""
        config = {
            'confluence': {
                'url': None,
                'username': None,
                'password': None,
                'space': None,
                'parent_page_id': None
            }
        }

        # 1. 尝试从配置文件加载（最低优先级）
        if config_path:
            try:
                with open(config_path, 'r', encoding='utf-8') as f:
                    file_config = yaml.safe_load(f)
                    if file_config and 'confluence' in file_config:
                        config['confluence'].update(file_config['confluence'])
            except Exception as e:
                print(f"⚠️ 警告: 无法读取配置文件 {config_path}: {str(e)}")
                print("将尝试使用环境变量...")

        # 2. 从环境变量加载（中等优先级）
        env_config = {
            'url': os.environ.get('KMS_URL'),
            'username': os.environ.get('KMS_USERNAME'),
            'password': os.environ.get('KMS_PASSWORD'),
            'space': os.environ.get('KMS_SPACE')
        }
        
        # 只更新非空的环境变量值
        config['confluence'].update({
            k: v for k, v in env_config.items() if v is not None
        })

        # 3. 从命令行参数加载（最高优先级）
        if cli_config and 'confluence' in cli_config:
            # 只更新非空的命令行参数值
            cli_values = cli_config['confluence']
            if cli_values:  # 确保有值才更新
                config['confluence'].update(cli_values)

        # 验证必要的配置项
        ConfigLoader._validate_config(config)
        
        return config

    @staticmethod
    def _validate_config(config):
        """验证配置是否完整"""
        required_keys = ['url', 'username', 'password', 'space']
        missing_keys = [
            key for key in required_keys 
            if not config['confluence'].get(key)
        ]
        
        if missing_keys:
            raise ValueError(
                f"缺少必要的配置项: {', '.join(missing_keys)}\n"
                "请通过以下方式之一提供配置:\n"
                "1. 命令行参数:\n"
                "   --url, --username, --password, --space\n"
                "2. 环境变量:\n"
                "   KMS_URL, KMS_USERNAME, KMS_PASSWORD, KMS_SPACE\n"
                "3. 配置文件中设置这些值"
            ) 