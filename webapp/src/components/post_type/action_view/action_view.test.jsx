import React from 'react';
import {shallow} from 'enzyme';

import ActionView from 'components/post_type/action_view/action_view';

describe('component/action_view/ActionView', () => {
    const samplePollId = 'samplepollid1';
    const baseProps = {
        post: {
            id: 'post_id',
            props: {
                poll_id: samplePollId,
            },
        },
        attachment: {
            actions: [
                {id: 'action_id1', name: 'answer1', type: 'button'},
                {id: 'action_id2', name: 'answer2', type: 'button'},
                {id: 'action_id3', name: 'answer3', type: 'button'},
            ],
        },
        votedAnswers: {
            samplepollid1: ['answer1', 'answer2'],
        },
        siteUrl: 'http://localhost:8065',
        actions: {
            fetchVotedAnswers: jest.fn(),
        },
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<ActionView {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot without any actions', () => {
        const newProps = {
            ...baseProps,
            attachment: {actions: []},
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with only select actions', () => {
        const newProps = {
            ...baseProps,
            attachment: {
                actions: [
                    {id: 'action_id1', name: 'answer1', type: 'button'},
                    {id: 'action_id2', name: 'answer2', type: 'select'},
                ],
            },
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
