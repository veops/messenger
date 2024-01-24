import { useEffect, useState, useRef } from 'react';
import { Table, Space, Message, Input, DatePicker } from '@arco-design/web-react';
import { IconSearch } from '@arco-design/web-react/icon';
import axios from 'axios';

import './App.css';

const { RangePicker } = DatePicker

function App() {
  const senderInputSearch = useRef(null)
  const contentInputSearch = useRef(null)
  const timeRangeSearch = useRef(null)

  const columns = [
    {
      title: 'Id',
      dataIndex: 'id',
      width: 100,
    },
    {
      title: '通知方式',
      dataIndex: 'sender',
      filterIcon: <IconSearch />,
      filterDropdown: ({ filterKeys, setFilterKeys, confirm }) => {
        return (
          <div className='arco-table-custom-filter'>
            <Input.Search
              ref={senderInputSearch}
              searchButton
              placeholder='请输入消息通知方式'
              value={filterKeys[0] || ''}
              onChange={(value) => {
                setFilterKeys(value ? [value] : []);
              }}
              onSearch={() => {
                confirm();
              }}
            />
          </div>
        );
      },
      onFilterDropdownVisibleChange: (visible) => {
        if (visible) {
          setTimeout(() => senderInputSearch.current.focus(), 150);
        }
      },
    },
    {
      title: '内容',
      dataIndex: 'content',
      filterIcon: <IconSearch />,
      filterDropdown: ({ filterKeys, setFilterKeys, confirm }) => {
        return (
          <div className='arco-table-custom-filter'>
            <Input.Search
              ref={contentInputSearch}
              searchButton
              placeholder='请输入消息内容'
              value={filterKeys[0] || ''}
              onChange={(value) => {
                setFilterKeys(value ? [value] : []);
              }}
              onSearch={() => {
                confirm();
              }}
            />
          </div>
        );
      },
      onFilterDropdownVisibleChange: (visible) => {
        if (visible) {
          setTimeout(() => contentInputSearch.current.focus(), 150);
        }
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      filters: [
        {
          text: '成功',
          value: true
        },
        {
          text: '失败',
          value: false
        },
      ]
    },
    {
      title: '消息接收时间',
      dataIndex: 'received_at',
    },
    {
      title: '消息发送时间',
      dataIndex: 'created_at',
      sorter: true,
      filterIcon: <IconSearch />,
      filterDropdown: ({ filterKeys, setFilterKeys, confirm }) => {
        return (
          <div className='arco-table-custom-filter'>
            <RangePicker
              ref={timeRangeSearch}
              showTime={{
                format: 'HH:mm',
              }}
              allowClear
              value={filterKeys || []}
              format='YYYY-MM-DD HH:mm'
              onOk={(value) => {
                confirm(value);
              }}
              onClear={() => {
                confirm([]);
              }}
            />
          </div>
        );
      },
    }
  ];

  const [data, setData] = useState([]);
  const [pagination, setPagination] = useState({
    sizeCanChange: true,
    showTotal: true,
    total: 0,
    pageSize: 10,
    current: 1,
    pageSizeChangeResetCurrent: true,
  });
  const [loading, setLoading] = useState(false);

  function onChangeTable(pagination, sorter, filters) {
    const { current, pageSize } = pagination;
    setLoading(true);
    setTimeout(() => {
      let params = {
        page_index: current,
        page_size: pageSize
      }
      if (filters?.sender?.length) {
        params.sender = filters.sender[0]
      }
      if (filters?.content?.length) {
        params.content = filters.content[0]
      }
      if (filters?.status?.length) {
        params.status = filters.status.join()
      }
      if (filters?.created_at?.length) {
        params.start = new Date(filters.created_at[0]).getTime() / 1000
        params.end = new Date(filters.created_at[1]).getTime() / 1000
      }
      if (sorter?.direction) {
        params.sort = (sorter.direction === 'ascend' ? '+' : '-') + sorter.field
      }

      axios.get('/v1/histories',
        {
          params: params
        }).then((response) => {
          const data = response.data?.list.map(item => {
            let tm = new Date(item.received_at * 1000)
            item.received_at = tm.toLocaleString()
            tm = new Date(item.created_at * 1000)
            item.created_at = tm.toLocaleString()
            item.status = item.status ? '成功' : '失败'
            const msg = JSON.parse(item.message)
            item.sender = msg.sender
            item.content = msg.content
            item.key = item.id
            return item
          })
          setData(data);
          const total = response.data?.count
          setPagination((pagination) => ({ ...pagination, current, pageSize, total }));
        }).catch(e => {
          Message.error("请求错误:(")
        })
      setLoading(false);
    }, 1000);
  }

  function expandedRowRender(record) {
    const columns = [
      {
        title: '参数',
        dataIndex: 'arg',
        width: 100,
      },
      {
        title: '详情',
        dataIndex: 'detial',
      },
    ]
    const dt = [
      {
        arg: "消息",
        detial: record.message
      },
      {
        arg: "发送请求",
        detial: record.req
      },
      {
        arg: "请求响应",
        detial: record.resp
      },
      {
        arg: "错误详情",
        detial: record.err
      },
    ]
    return (
      <Table columns={columns} data={dt} pagination={false} />
    );
  };


  useEffect(() => {
    onChangeTable(pagination)
  }, [])

  return (
    <Table
      className='table-demo-resizable-column'
      stripe
      border
      borderCell
      columns={columns}
      loading={loading}
      data={data}
      pagination={pagination}
      expandedRowRender={expandedRowRender}
      indentSize={75}
      onChange={onChangeTable}
      expandProps={{
        width: 60,
        expandRowByClick: true,
      }}
      renderPagination={(paginationNode) => (
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            marginTop: 10,
          }}
        >
          <Space></Space>
          {paginationNode}
        </div>
      )}
    />
  );
}

export default App;

