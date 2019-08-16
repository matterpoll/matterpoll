import ActionTypes from 'action_types';
import {postTypeComponent} from 'reducers/post_type';

describe('post_type reducers', () => {
    test('no action', () => expect(postTypeComponent(undefined, {})).toEqual({})); // eslint-disable-line no-undefined
    test('no action with initial state', () => {
        expect(
            postTypeComponent({id: 'component_id'}, {})
        ).toEqual({id: 'component_id'});
    });
    test('action type without data', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGIST_POST_TYPE_COMPONENT_ID, data: undefined}), // eslint-disable-line no-undefined
        ).toEqual({id: 'component_id'});
    });
    test('action type without postTypeComponentId', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGIST_POST_TYPE_COMPONENT_ID, data: {dummy: 'id'}}),
        ).toEqual({id: 'component_id'});
    });
    test('action with component_id', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGIST_POST_TYPE_COMPONENT_ID, data: {postTypeComponentId: 'new_component_id'}})
        ).toEqual({id: 'new_component_id'});
    });
    test('action with empty id', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGIST_POST_TYPE_COMPONENT_ID, data: {postTypeComponentId: ''}})
        ).toEqual({id: ''});
    });
});
