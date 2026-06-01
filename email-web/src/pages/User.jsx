import { useEffect, useState } from 'react'

import { Button, Form, Input, Layout, Message, Modal, Radio, Space, Table } from '@arco-design/web-react'
import { IconPlus } from '@arco-design/web-react/icon'

import request from 'src/api/request'

const suffix = import.meta.env.VITE_SUFFIX

const UserPage = () => {
  const [formEdit] = Form.useForm()
  const [tableData, setTableData] = useState({})

  const [visible, setVisible] = useState(false)
  const [editUser, setEditUser] = useState(null)

  const [pageHeight, setPageHeight] = useState(0)

  const columns = [
    {
      title: '姓名',
      dataIndex: 'full_name',
      record: (_, record) => record?.person_first_name + record?.person_last_name,
    },
    {
      title: '账号',
      dataIndex: 'email',
    },
    {
      title: '管理员',
      dataIndex: 'is_admin',
      align: 'center',
      render: (is_admin) => (is_admin ? '是' : '否'),
    },
    {
      title: '最后登录时间',
      dataIndex: 'last_logon_time',
    },
    {
      title: '操作',
      dataIndex: 'action',
      align: 'center',
      render: (_, record) => (
        <Space>
          <Button type='text' size='mini' onClick={() => EditUser(record)}>
            编辑
          </Button>
          <Button type='text' size='mini' status='danger' onClick={() => DeleteUser(record)}>
            删除
          </Button>
        </Space>
      ),
    },
  ]
  // 提交
  const Submit = async () => {
    const values = await formEdit.validate()
    const params = {
      email: values.email + suffix,
      person_first_name: values.person_first_name,
      person_last_name: values.person_last_name,
      is_admin: values.is_admin,
    }

    let url = '/api/user/update'
    if (!editUser?.id) {
      url = '/api/user/create'
      params.password = values.password
    } else {
      params.id = editUser.id
    }

    const { code, msg } = await request.post(url, params)
    if (code === 200) {
      Message.success(msg)
      getUserList()
    } else {
      Message.error(msg)
    }

    setVisible(false)
    setEditUser(null)
  }

  // 编辑用户
  const EditUser = async (record) => {
    setEditUser(record)
    setVisible(true)
    formEdit.resetFields()

    if (record?.id) {
      const old = {
        ...record,
        email: record?.email?.split('@')[0],
      }
      formEdit.setFieldsValue(old)
    } else {
      formEdit.setFieldsValue({ is_admin: 0 })
    }
  }

  // 删除用户
  const DeleteUser = async (record) => {
    Modal.confirm({
      title: '警告',
      content: `请确认是否删除 ${record.email}?`,
      className: 'simpleModal',
      onOk: async () => {
        const { code, msg } = await request.post('/api/user/delete', { email: record.email })
        if (code === 200) {
          Message.success(msg)
          getUserList()
        } else {
          Message.error(msg)
        }
      },
    })
  }
  // 获取用户列表
  const getUserList = async () => {
    const { code, data, msg } = await request.post('/api/user/list')
    if (code === 200) {
      setTableData(data)
    } else {
      Message.error(msg)
    }
  }

  // 获取页面高度
  const getPageHeight = () => {
    const height = document.body.clientHeight - 160
    setPageHeight(height)
  }

  useEffect(() => {
    window.addEventListener('resize', getPageHeight())
    getPageHeight()

    getUserList()
  }, [])

  return (
    <Layout.Content className={'p-4 bg-white mx-4 rounded-xl'}>
      <div className='mb-4 flex items-center justify-between'>
        <div className='text-lg font-bold'>用户管理</div>
        <Button type='primary' icon={<IconPlus />} onClick={() => EditUser()}>
          新增用户
        </Button>
      </div>
      <Table rowKey={'id'} columns={columns} data={tableData?.list || []} pagination={false} scroll={{ y: pageHeight }} />

      <Modal
        title={editUser?.id ? '编辑用户' : '新增用户'}
        visible={visible}
        onOk={() => Submit()}
        onCancel={() => setVisible(false)}>
        <Form form={formEdit} autoComplete='off' layout='vertical'>
          <Form.Item label='账号' field='email' rules={[{ required: true, message: '请输入账号' }]}>
            <Input suffix={suffix} placeholder='请输入账号' />
          </Form.Item>
          <Form.Item
            label='密码'
            field='password'
            hidden={editUser?.id}
            rules={[{ required: !editUser?.id, message: '请输入密码' }]}>
            <Input.Password placeholder='请输入密码' />
          </Form.Item>
          <Form.Item label='姓' field='person_first_name' rules={[{ required: true, message: '请输入姓' }]}>
            <Input placeholder='请输入姓' />
          </Form.Item>
          <Form.Item label='名' field='person_last_name' rules={[{ required: true, message: '请输入名' }]}>
            <Input placeholder='请输入名' />
          </Form.Item>
          <Form.Item label='管理员' field='is_admin'>
            <Radio.Group>
              <Radio value={0}>否</Radio>
              <Radio value={1}>是</Radio>
            </Radio.Group>
          </Form.Item>
        </Form>
      </Modal>
    </Layout.Content>
  )
}
export default UserPage
