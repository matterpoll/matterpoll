import ActionTypes from '@/action_types';
import {postTypeComponent} from '@/reducers/post_type';

// Every other action on the webapp's store reaches this reducer too, and must leave it alone.
const unrelatedAction = {type: 'unrelated_action'};

describe('post_type reducers', () => {
    test('no action', () => expect(postTypeComponent(undefined, unrelatedAction)).toEqual({})); // eslint-disable-line no-undefined
    test('no action with initial state', () => {
        expect(
            postTypeComponent({id: 'component_id'}, unrelatedAction),
        ).toEqual({id: 'component_id'});
    });
    test('action type without data', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGISTER_POST_TYPE_COMPONENT_ID, data: undefined}), // eslint-disable-line no-undefined
        ).toEqual({id: 'component_id'});
    });
    test('action type without postTypeComponentId', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGISTER_POST_TYPE_COMPONENT_ID, data: {}}),
        ).toEqual({id: 'component_id'});
    });
    test('action with component_id', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGISTER_POST_TYPE_COMPONENT_ID, data: {postTypeComponentId: 'new_component_id'}}),
        ).toEqual({id: 'new_component_id'});
    });
    test('action with empty id', () => {
        expect(
            postTypeComponent(
                {id: 'component_id'},
                {type: ActionTypes.REGISTER_POST_TYPE_COMPONENT_ID, data: {postTypeComponentId: ''}}),
        ).toEqual({id: ''});
    });
});
