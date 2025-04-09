import os
import yaml

class ConfigLoader:
    """配置加载器，处理配置文件和环境变量"""

    @staticmethod
    def load_config(config_path=None):
        """加载配置文件或环境变量"""
        config = {
            'confluence': {
                'url': None,
                'username': None,
                'password': None,
                'space': None,
                'parent_page_id': None
            }
        }

        # 如果指定了配置文件，从文件加载
        if config_path:
            try:
                with open(config_path, 'r', encoding='utf-8') as f:
                    file_config = yaml.safe_load(f)
                    if file_config and 'confluence' in file_config:
                        config.update(file_config)
            except Exception as e:
                print(f"⚠️ 警告: 无法读取配置文件 {config_path}: {str(e)}")
                print("将尝试使用环境变量...")
        
        # 如果没有指定配置文件或配置文件加载失败，尝试从环境变量读取
        if not config_path or not all([
            config['confluence']['url'],
            config['confluence']['username'],
            config['confluence']['password'],
            config['confluence']['space']
        ]):
            config['confluence'].update({
                'url': os.environ.get('KMS_URL'),
                'username': os.environ.get('KMS_USERNAME'),
                'password': os.environ.get('KMS_PASSWORD'),
                'space': os.environ.get('KMS_SPACE')
            })

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
                "请在配置文件中设置这些值，或设置对应的环境变量:\n"
                "KMS_URL, KMS_USERNAME, KMS_PASSWORD, KMS_SPACE"
            ) 