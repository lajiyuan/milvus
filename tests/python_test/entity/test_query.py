import logging

import pytest
from pymilvus import DataType

import utils as ut

default_entities = ut.gen_entities(ut.default_nb, is_normal=True)
raw_vectors, default_binary_entities = ut.gen_binary_entities(ut.default_nb)
default_int_field_name = "int64"
default_float_field_name = "float"
default_term_expr = f'{default_int_field_name} in [0, 1]'


def init_data(connect, collection, nb=ut.default_nb, partition_names=None, auto_id=True):
    """
    Generate entities and add it in collection
    """
    if nb == 3000:
        insert_entities = default_entities
    else:
        insert_entities = ut.gen_entities(nb, is_normal=True)
    if partition_names is None:
        if auto_id:
            ids = connect.insert(collection, insert_entities)
        else:
            ids = connect.insert(collection, insert_entities, ids=[i for i in range(nb)])
    else:
        if auto_id:
            ids = connect.insert(collection, insert_entities, partition_name=partition_names)
        else:
            ids = connect.insert(collection, insert_entities, ids=[i for i in range(nb)],
                                 partition_name=partition_names)
    connect.flush([collection])
    return insert_entities, ids


def init_binary_data(connect, collection, nb=3000, insert=True, partition_names=None):
    """
    Generate entities and add it in collection
    """
    ids = []
    # global binary_entities
    global raw_vectors
    if nb == 3000:
        insert_entities = default_binary_entities
        insert_raw_vectors = raw_vectors
    else:
        insert_raw_vectors, insert_entities = ut.gen_binary_entities(nb)
    if insert is True:
        if partition_names is None:
            ids = connect.insert(collection, insert_entities)
        else:
            ids = connect.insert(collection, insert_entities, partition_name=partition_names)
        connect.flush([collection])
    return insert_raw_vectors, insert_entities, ids


@pytest.mark.skip(reason="waiting for collection update")
class TestQueryBase:
    """
    test Query interface
    query(collection_name, expr, output_fields=None, partition_names=None, timeout=None)
    """

    @pytest.fixture(
        scope="function",
        params=ut.gen_invalid_strs()
    )
    def get_collection_name(self, request):
        yield request.param

    @pytest.fixture(
        scope="function",
        params=ut.gen_simple_index()
    )
    def get_simple_index(self, request, connect):
        return request.param

    def test_query(self, connect, collection):
        """
        target: test query
        method: query with term expr
        expected: verify query result
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        pos = 5
        connect.load_collection(collection)
        term_expr = f'{default_int_field_name} in {entities[:pos]}'
        res = connect.query(collection, term_expr)
        logging.getLogger().debug(res)

    def test_query_collection_not_existed(self, connect):
        """
        target: test query not existed collection
        method: query not existed collection
        expected: raise exception
        """
        ex_msg = 'can not found collection'
        collection = "not_exist"
        with pytest.raises(Exception, match=ex_msg):
            connect.query(collection, default_term_expr)

    def test_query_without_connect(self, dis_connect, collection):
        """
        target: test query without connection
        method: close connect and query
        expected: raise exception
        """
        ex_msg = 'todo'
        with pytest.raises(Exception, match=ex_msg):
            dis_connect.query(collection, default_term_expr)

    def test_query_invalid_collection_name(self, connect, get_collection_name):
        """
        target: test query with invalid collection name
        method: query with invalid collection name
        expected: raise exception
        """
        collection_name = get_collection_name
        msg = 'invalid collection name'
        with pytest.raises(Exception, match=msg):
            connect.query(collection_name, default_term_expr)

    def test_query_after_index(self, connect, collection, get_simple_index):
        """
        target: test query after creating index
        method: query after index
        expected: query result is correct
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.create_index(collection, ut.default_float_vec_field_name, get_simple_index)
        connect.load_collection(collection)
        res = connect.query(collection, default_term_expr)
        logging.getLogger().info(res)

    def test_query_after_search(self, connect, collection):
        """
        target: test query after search
        method: query after search
        expected: query result is correct
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        top_k = 10
        nq = 2
        query, _ = ut.gen_query_vectors(ut.default_float_vec_field_name, entities, top_k=top_k, nq=nq)
        connect.load_collection(collection)
        search_res = connect.search(collection, query)
        assert len(search_res) == nq
        assert len(search_res[0]) == top_k
        query_res = connect.query(collection, default_term_expr)
        logging.getLogger().info(query_res)

    def test_query_empty_collection(self, connect, collection):
        """
        target: test query empty collection
        method: query on a empty collection
        expected: todo
        """
        connect.load_collection(collection)
        res = connect.query(collection, default_term_expr)
        logging.getLogger().info(res)

    def test_query_without_loading(self, connect, collection):
        """
        target: test query without loading
        method: no loading before query
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        msg = 'can not find collection'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, default_term_expr)

    def test_query_auto_id_collection(self, connect, id_collection):
        """
        target: test query on collection that primary field auto_id=True
        method: 1.create collection with auto_id=True 2.query on primary field
        expected: todo
        """
        pass

    def test_query_expr_none(self, connect, collection):
        """
        target: test query with none expr
        method: query with expr None
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, None)

    @pytest.mark.parametrize("expr", [1, 2., [], {}, ()])
    def test_query_expr_non_string(self, connect, collection, expr):
        """
        target: test query with non-string expr
        method: query with non-string expr, eg 1, [] ..
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, expr)

    @pytest.mark.parametrize("expr", ["12-s", "中文", "a", " "])
    def test_query_expr_invalid_string(self, connect, collection, expr):
        """
        target: test query with invalid expr
        method: query with invalid string expr
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, expr)

    def test_query_expr_term(self, connect, collection):
        """
        target: test query with TermExpr
        method: query with TermExpr
        expected: query result is correct
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        res = connect.query(collection, default_term_expr)
        logging.getLogger().info(res)

    def test_query_expr_not_existed_field(self, connect, collection):
        """
        target: test query with not existed field
        method: query by term expr with fake field
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        term_expr = 'field in [1, 2]'
        msg = 'field not existed'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, term_expr)

    def test_query_expr_unsupported_field(self, connect, collection):
        """
        target: test query on unsupported field
        method: query on float field
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        term_expr = f'{default_float_field_name} in [1., 2.]'
        msg = 'only supported on int field'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, term_expr)

    def test_query_expr_non_primary_field(self, connect, collection):
        """
        target: test query on non-primary field
        method: query on non-primary int field
        expected: raise exception
        """
        field_name = "int2"
        fields = ut.add_field_default(field_name=field_name)
        c_name = ut.gen_unique_str()
        connect.create_collection(c_name, fields)
        entities = ut.add_field(field_name=field_name)
        connect.insert(collection, entities)
        connect.flush(c_name)
        term_expr = f'{field_name} in [1, 2]'
        msg = 'only supported on primary field'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, term_expr)

    @pytest.mark.parametrize("expr", [f'{default_int_field_name} inn [1, 2]',
                                      f'{default_int_field_name} not in [1, 2]',
                                      f'{default_int_field_name} in not [1, 2]'])
    def test_query_expr_wrong_term_keyword(self, connect, collection, expr):
        """
        target: test query with wrong term expr keyword
        method: query with wrong keyword term expr
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, expr)

    @pytest.mark.parametrize("expr", [f'{default_int_field_name} in 1',
                                      f'{default_int_field_name} in "in"',
                                      f'{default_int_field_name} in (mn)'])
    def test_query_expr_non_array_term(self, connect, collection, expr):
        """
        target: test query with non-array term expr
        method: query with non-array term expr
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, expr)

    def test_query_expr_empty_term_array(self, connect, collection):
        """
        target: test query with empty array term expr
        method: query with empty term expr
        expected: todo
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        term_expr = f'{default_int_field_name} in []'
        res = connect.query(collection, term_expr)
        assert len(res) == 0

    def test_query_expr_single_term_array(self, connect, collection):
        """
        target: test query with single array term expr
        method: query with single array value
        expected: query result is one entity
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        term_expr = f'{default_int_field_name} in [0]'
        res = connect.query(collection, term_expr)
        assert len(res) == 1
        assert res[0][default_int_field_name] == entities[0]["values"][0]
        assert res[1][default_float_field_name] == entities[1]["values"][0]
        assert res[2][ut.default_float_vec_field_name] == entities[2]["values"][0]

    def test_query_binary_expr_single_term_array(self, connect, binary_collection):
        """
        target: test query with single array term expr
        method: query with single array value
        expected: query result is one entity
        """
        _, binary_entities, ids = init_binary_data(connect, binary_collection)
        assert len(ids) == ut.default_nb
        connect.flush(binary_collection)
        term_expr = f'{default_int_field_name} in [0]'
        res = connect.query(binary_collection, term_expr)
        assert len(res) == 1
        assert res[0][default_int_field_name] == binary_entities[0]["values"][0]
        assert res[1][default_float_field_name] == binary_entities[1]["values"][0]
        assert res[2][ut.default_float_vec_field_name] == binary_entities[2]["values"][0]

    def test_query_expr_all_term_array(self, connect, collection):
        """
        target: test query with all array term expr
        method: query with all array value
        expected: verify query result
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        int_values = entities[0]["values"]
        term_expr = f'{default_int_field_name} in {int_values}'
        res = connect.query(collection, term_expr)
        assert len(res) == ut.default_nb
        for i in ut.default_nb:
            assert res[i][default_int_field_name] == int_values[i]

    def test_query_expr_half_term_array(self, connect, collection):
        """
        target: test query with half array term expr
        method: query with half array value
        expected: verify query result
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        half = ut.default_nb // 2
        connect.flush(collection)
        int_values = entities[0]["values"][:half]
        term_expr = f'{default_int_field_name} in {int_values}'
        res = connect.query(collection, term_expr)
        assert len(res) == half
        for i in half:
            assert res[i][default_int_field_name] == entities[0]["values"][i]
            assert res[i][default_float_field_name] == entities[1]["values"][i]
            assert res[i][ut.default_float_vec_field_name] == entities[2]["values"][i]

    def test_query_expr_repeated_term_array(self, connect, collection):
        """
        target: test query with repeated term array on primary field with unique value
        method: query with repeated array value
        expected: verify query result
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        int_values = [0, 0]
        term_expr = f'{default_int_field_name} in {int_values}'
        res = connect.query(collection, term_expr)
        assert len(res) == 1

    def test_query_expr_inconstant_term_array(self, connect, collection):
        """
        target: test query with term expr that field and array are inconsistent
        method: query with int field and float values
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        expr = f'{default_int_field_name} in [1., 2.]'
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, expr)

    def test_query_expr_mix_term_array(self, connect, collection):
        """
        target: test query with mix type value expr
        method: query with term expr that has int and float type value
        expected: todo
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        expr = f'{default_int_field_name} in [1, 2.]'
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, expr)

    @pytest.mark.parametrize("constant", [[1], (), {}, " "])
    def test_query_expr_non_constant_array_term(self, connect, collection, constant):
        """
        target: test query with non-constant array term expr
        method: query with non-constant array expr
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        expr = f'{default_int_field_name} in [{constant}]'
        msg = 'invalid expr'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, expr)

    def test_query_output_field_none(self, connect, collection):
        """
        target: test query with none output field
        method: query with output field=None
        expected: return all fields
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        res = connect.query(collection, default_term_expr, output_fields=None)
        fields = [default_int_field_name, default_float_field_name, ut.default_float_vec_field_name]
        assert res[0].keys() == fields

    def test_query_output_one_field(self, connect, collection):
        """
        target: test query with output one field
        method: query with output one field
        expected: return one field
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        res = connect.query(collection, default_term_expr, output_fields=[default_int_field_name])
        assert res[0].keys() == [default_int_field_name]

    def test_query_output_all_fields(self, connect, collection):
        """
        target: test query with none output field
        method: query with output field=None
        expected: return all fields
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        fields = [default_int_field_name, default_float_field_name, ut.default_float_vec_field_name]
        res = connect.query(collection, default_term_expr, output_fields=fields)
        assert res[0].keys() == fields

    def test_query_output_not_existed_field(self, connect, collection):
        """
        target: test query output not existed field
        method: query with not existed output field
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'cannot find field'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, default_term_expr, output_fields=["int"])

    def test_query_output_part_not_existed_field(self, connect, collection):
        """
        target: test query output part not existed field
        method: query with part not existed field
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'cannot find field'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, default_term_expr, output_fields=[default_int_field_name, "int"])

    def test_query_empty_output_fields(self, connect, collection):
        """
        target: test query with empty output fields
        method: query with empty output fields
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'output fields is empty'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, default_term_expr, output_fields=[])

    @pytest.mark.parametrize("fields", ut.gen_invalid_strs())
    def test_query_invalid_output_fields(self, connect, collection, fields):
        """
        target: test query with invalid output fields
        method: query with invalid field fields
        expected: raise exception
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.flush(collection)
        msg = 'invalid output fields'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, default_term_expr, output_fields=fields)


@pytest.mark.skip(reason="waiting for collection update")
class TestQueryPartition:
    """
    test Query interface
    query(collection_name, expr, output_fields=None, partition_names=None, timeout=None)
    """

    def test_query_partition(self, connect, collection):
        """
        target: test query on partition
        method: create a partition and query
        expected: verify query result
        """
        connect.create_partition(collection, ut.default_tag)
        entities, ids = init_data(connect, collection, ut.default_tag)
        assert len(ids) == ut.default_nb
        connect.load_partitions(collection, [ut.default_tag])
        res = connect.query(collection, default_term_expr, partition_names=[ut.default_tag])
        # todo res

    def test_query_partition_without_loading(self, connect, collection):
        """
        target: test query on partition without loading
        method: query on partition and no loading
        expected: raise exception
        """
        connect.create_partition(collection, ut.default_tag)
        entities, ids = init_data(connect, collection, ut.default_tag)
        assert len(ids) == ut.default_nb
        msg = 'cannot find collection'
        with pytest.raises(Exception, match=msg):
            connect.query(collection, default_term_expr, partition_names=[ut.default_tag])

    def test_query_default_partition(self, connect, collection):
        """
        target: test query on default partition
        method: query on default partition
        expected: verify query result
        """
        entities, ids = init_data(connect, collection)
        assert len(ids) == ut.default_nb
        connect.load_partitions(collection, [ut.default_tag])
        res = connect.query(collection, default_term_expr, partition_names=[ut.default_partition_name])
        # todo res

    def test_query_empty_partition(self, connect, collection):
        """
        target: test query on empty partition
        method: query on a empty collection
        expected: empty query result
        """
        connect.create_partition(collection, ut.default_tag)
        connect.load_partitions(collection, [ut.default_tag])
        res = connect.query(collection, default_term_expr, partition_names=[ut.default_partition_name])
        assert len(res) == 0

    def test_query_not_existed_partition(self, connect, collection):
        """
        target: test query on a not existed partition
        method: query on not existed partition
        expected: raise exception
        """
        connect.load_partitions(collection, [ut.default_tag])
        msg = 'cannot find partition'
        tag = ut.gen_unique_str()
        with pytest.raises(Exception, match=msg):
            connect.query(collection, default_term_expr, partition_names=[tag])

    def test_query_partition_repeatedly(self, connect, collection):
        """
        target: test query repeatedly on partition
        method: query on partition twice
        expected: verify query result
        """
        connect.create_partition(collection, ut.default_tag)
        entities, ids = init_data(connect, collection, ut.default_tag)
        assert len(ids) == ut.default_nb
        connect.load_partitions(collection, [ut.default_tag])
        res_one = connect.query(collection, default_term_expr, partition_names=[ut.default_tag])
        res_two = connect.query(collection, default_term_expr, partition_names=[ut.default_tag])
        assert res_one == res_two

    def test_query_another_partition(self, connect, collection):
        """
        target: test query another partition
        method: 1. insert entities into two partitions
                2.query on one partition and query result empty
        expected: query result is empty
        """
        insert_entities_into_two_partitions_in_half(connect, collection)
        half = ut.default_nb // 2
        term_expr = f'{default_int_field_name} in [{half}]'
        res = connect.query(collection, term_expr, partition_names=[ut.default_tag])
        assert len(res) == 0

    def test_query_multi_partitions_multi_results(self, connect, collection):
        """
        target: test query on multi partitions and get multi results
        method: 1.insert entities into two partitions
                2.query on two partitions and query multi result
        expected: query results from two partitions
        """
        entities, entities_2 = insert_entities_into_two_partitions_in_half(connect, collection)
        half = ut.default_nb // 2
        term_expr = f'{default_int_field_name} in [{half - 1}, {half}]'
        res = connect.query(collection, term_expr, partition_names=[ut.default_tag, ut.default_partition_name])
        assert len(res) == 2
        assert res[0][default_int_field_name] == entities[0]["values"][-1]
        assert res[1][default_int_field_name] == entities_2[0]["values"][0]

    def test_query_multi_partitions_single_result(self, connect, collection):
        """
        target: test query on multi partitions and get single result
        method: 1.insert into two partitions
                2.query on two partitions and query single result
        expected: query from two partitions and get single result
        """
        entities, entities_2 = insert_entities_into_two_partitions_in_half(connect, collection)
        half = ut.default_nb // 2
        term_expr = f'{default_int_field_name} in [{half}]'
        res = connect.query(collection, term_expr, partition_names=[ut.default_tag, ut.default_partition_name])
        assert len(res) == 1
        assert res[0][default_int_field_name] == entities_2[0]["values"][0]


def insert_entities_into_two_partitions_in_half(connect, collection):
    """
    insert default entities into two partitions(default_tag and _default) in half(int64 and float fields values)
    :param connect: milvus connect
    :param collection: milvus created collection
    :return: entities of default_tag and entities_2 of _default
    """
    connect.create_partition(collection, ut.default_tag)
    half = ut.default_nb // 2
    entities, _ = init_data(connect, collection, nb=half, partition_names=ut.default_tag)
    vectors = ut.gen_vectors(half, ut.default_dim)
    entities_2 = [
        {"name": "int64", "type": DataType.INT64, "values": [i for i in range(half, ut.default_nb)]},
        {"name": "float", "type": DataType.FLOAT, "values": [float(i) for i in range(half, ut.default_nb)]},
        {"name": ut.default_float_vec_field_name, "type": DataType.FLOAT_VECTOR, "values": vectors}
    ]
    connect.insert(collection, entities_2)
    connect.flush([collection])
    connect.load_collection(collection)
    return entities, entities_2
