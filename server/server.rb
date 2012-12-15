require 'rubygems'
require 'bundler'
require 'securerandom'
require 'json'
Bundler.require

API_KEY = "AIzaSyDN9vs2KvUeDeHqAWT56CqnriFhvZCvL-8"
REGISTRATION_IDS = "registration_ids"
STORE = Redis.new

get '/register/:registration_id' do
  registration_ids = JSON.parse(STORE.get(REGISTRATION_IDS) || "[]")
  current_registration_id = params[:registration_id]
  registration_ids << current_registration_id unless registration_ids.include?(current_registration_id)
  STORE.set(REGISTRATION_IDS, registration_ids.to_json)
end

get '/push' do
  registration_ids = JSON.parse(STORE.get(REGISTRATION_IDS))
  gcm = GCM.new(API_KEY)
  message_id = SecureRandom.uuid
  options = {
      :data => {
          :id => message_id
      },
      :collapse_key => "message"
  }
  STORE.set(message_id, {:start => Time.now}.to_json)
  response = gcm.send_notification(registration_ids, options)
  [response[:status], response[:headers], response[:body]]
end


get '/received/:message_id' do
  message_id = params[:message_id]
  stats = JSON.parse(STORE.get(message_id))
  stats[:end] = Time.now
  STORE.set(message_id, stats.to_json)
end
