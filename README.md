# jna-nuh-yoh-guh
ज्ञान योग

Jnana Yoga is one of the paths to Moksha in Hindu Philosophy

I personally also believe that the path to knowledge does free you of a lot of your constructs, though not all because all perception is itself a process of creating your own construct.

Through this project I aim to model my own knowledge, find inconsistencies, track questions that I am struggling with and find answers with the help of the amazing tools technology has made available to us.

I was trying to find the best or perfect way to accomplish this, but I suddenly remember how as practictioners of science, we have to believe that there might be something wrong with our hypothesis, I knew that I need to start working on this project rather than trying to have the perfect knowledge for it. But falibilism answered the "Why".

I need to make changes to the way I have modelled data in this project I have settled on the systems way of thinking about his. This is the new schema:
Nodes:
- Measurment
- System
- Stock
- Flow

Relationships:
- CONTAINS (System - System)
- HAS_STOCK (System - Stock)
- DRAINS (Flow - Stock) (Flow - Flow)
- FILLS (Flow - Stock) (Flow - Flow)

Let's assume Flows affect Flows through some intermediary stock
