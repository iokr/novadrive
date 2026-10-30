import request from './request'

// 侧边栏空间条：{ used, quota, file_count }
export const storage = () => request.get('/user/storage')

// 个人中心空间分析：{ used, quota, total_size, total_count, dir_count, by_type }
export const overview = () => request.get('/user/overview')
