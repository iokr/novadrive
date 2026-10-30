import request from './request'

// 注意：登录 / 注册接口不携带旧 token 也能正常工作（后端不校验）

export const register = (payload) => request.post('/auth/register', payload)

export const login = (payload) => request.post('/auth/login', payload)

export const profile = () => request.get('/auth/profile')

export const updateProfile = (payload) => request.put('/auth/profile', payload)

export const changePassword = (payload) => request.put('/auth/password', payload)
