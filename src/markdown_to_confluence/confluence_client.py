from atlassian import Confluence
import urllib3
import warnings

# 忽略 urllib3 的 SSL 警告
warnings.filterwarnings('ignore', category=urllib3.exceptions.InsecureRequestWarning)

class ConfluenceClient:
    """Confluence 客户端，处理与 Confluence 的交互"""

    def __init__(self, config):
        """初始化 Confluence 客户端"""
        self.config = config
        self.client = self._init_confluence()

    def _init_confluence(self):
        """初始化 Confluence 连接"""
        try:
            return Confluence(
                url=self.config['confluence']['url'],
                username=self.config['confluence']['username'],
                password=self.config['confluence']['password'],
                verify_ssl=False
            )
        except Exception as e:
            raise Exception(f"连接 Confluence 失败: {str(e)}")

    def find_page_in_parent(self, title, parent_page_id):
        """在指定父页面下查找页面"""
        try:
            children = self.client.get_child_pages(parent_page_id)
            for child in children:
                if child['title'] == title:
                    return child
            return None
        except Exception as e:
            print(f"警告: 查找页面失败: {str(e)}")
            return None

    def get_page_version(self, page_id):
        """获取页面的当前版本信息"""
        try:
            page = self.client.get_page_by_id(page_id)
            return page.get('version', {}).get('number', 0)
        except:
            return 0

    def update_page(self, page_id, title, body, space_key):
        """更新现有页面"""
        try:
            current_page = self.client.get_page_by_id(
                page_id=page_id,
                expand='version'
            )
            
            body_data = {
                'id': page_id,
                'type': 'page',
                'title': title,
                'space': {'key': space_key},
                'body': {
                    'storage': {
                        'value': body,
                        'representation': 'storage'
                    }
                },
                'version': {
                    'number': current_page['version']['number'] + 1
                }
            }
            
            result = self.client.put(
                f'/rest/api/content/{page_id}',
                data=body_data
            )
            
            if result:
                print(f"✅ 已成功更新页面: {title}")
                print(f"🔗 页面链接: {self.config['confluence']['url']}/pages/viewpage.action?pageId={page_id}")
                return True
            else:
                raise Exception("更新页面失败，API 返回为空")
                
        except Exception as e:
            print(f"⚠️ 更新页面时出错: {str(e)}")
            raise e

    def create_page(self, title, body, parent_page_id):
        """创建新页面"""
        try:
            new_page = self.client.create_page(
                space=self.config['confluence']['space'],
                title=title,
                body=body,
                parent_id=parent_page_id,
                type='page',
                representation='storage'
            )
            print(f"✅ 已成功创建页面: {title}")
            return new_page
        except Exception as e:
            print(f"⚠️ 创建页面时出错: {str(e)}")
            raise e

    def attach_content(self, content, name, content_type, page_id):
        """上传附件到页面"""
        return self.client.attach_content(
            content=content,
            name=name,
            content_type=content_type,
            page_id=page_id
        ) 